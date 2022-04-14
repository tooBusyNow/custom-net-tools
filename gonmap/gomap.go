package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mpvl/unique"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var (
	reset = "\033[0m"
	green = "\033[32m"
	red   = "\033[31m"
)

var top20 = "21-23,25,53,80,110-111,135,139,143,443,445,993,995,1723,3306,3389,5900,8080"

func portParseError() {
	fmt.Println("Can't parse port number")
	os.Exit(0)
}

func count_port_tabs(port int) string {
	if len(strconv.Itoa(port)) > 3 {
		return "\t"
	}
	return "\t\t"
}

func print_opened(port int, proto string) {
	out := fmt.Sprintf("|\t%d/%s ⟸\t| %s\t |\n", port, proto, "Opened ✔")
	fmt.Print(string(green), out, string(reset))
}

func print_closed(port int, proto string) {

	tabs := count_port_tabs(port)

	out := fmt.Sprintf("|\t%d/%s%s| %s\t |\n", port, proto, tabs, "Closed ✖")
	fmt.Print(string(red), out, string(reset))
}

func ports_parse(portRange string) []string {

	rawRange := strings.Split(portRange, ",")
	var scanPorts []string

	for _, raw_port := range rawRange {
		if strings.Contains(raw_port, "-") {
			splitted := strings.Split(raw_port, "-")

			s_port, err1 := strconv.Atoi(splitted[0])
			f_port, err2 := strconv.Atoi(splitted[1])

			if err1 != nil || err2 != nil || f_port < s_port {
				portParseError()
			}

			for i := s_port; i <= f_port; i++ {
				scanPorts = append(scanPorts, strconv.Itoa(i))
			}

		} else {
			if _, err := strconv.Atoi(raw_port); err != nil {
				portParseError()
			} else {
				scanPorts = append(scanPorts, raw_port)
			}
		}
	}

	sort.StringSlice(scanPorts).Sort()
	unique.Strings(&scanPorts)

	return scanPorts
}

func init_parse() (ports []string, ip net.IP, mTh bool) {

	var portRange string
	flag.StringVar(&portRange, "p", top20, "specifies port range to scan")

	var mThread bool
	flag.BoolVar(&mThread, "mth", false, "allows to run with goroutines (in parallel)")

	flag.Parse()
	if ip := net.ParseIP(flag.Arg(0)); ip != nil {

		scanPorts := ports_parse(portRange)
		return scanPorts, ip, mThread

	} else {
		fmt.Println("IP address is not provided or invalid")
		os.Exit(0)
	}
	return
}

func check_availability(ip net.IP) {

	fmt.Print("Trying to reach host... ")

	conn, _ := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	defer conn.Close()

	//// Create ICMP Echo message
	icmpMsg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("Hello There!"),
		},
	}

	/// Serialize ICMP message
	echoReq, err := icmpMsg.Marshal(nil)
	if err != nil {
		fmt.Println("Error occured during serialization")
		os.Exit(0)
	}

	if _, err := conn.WriteTo(echoReq, &net.IPAddr{IP: ip}); err != nil {
		fmt.Println("Something went wrong during writing to connection channel", ip)
		os.Exit(0)
	}

	/// Wait for ICMP reply from target
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	readBuff := make([]byte, 1500)
	n, _, err := conn.ReadFrom(readBuff)

	if err != nil {
		fmt.Printf("Target host is unreachable: %s", ip.String())
		os.Exit(0)
	}

	/// Parse ICMP reply
	rm, err := icmp.ParseMessage(1, readBuff[:n])
	if err != nil {
		fmt.Println("Unable to parse response ICMP message")
		os.Exit(0)
	}

	if rm.Type == ipv4.ICMPTypeEchoReply {
		fmt.Println("Succesfully got ICMP Echo Reply, ready to scan now!")
	} else {
		fmt.Println("Incorrect type of ICMP reply")
		os.Exit(0)
	}
}

func scan_tcp_port(socket string, port int, mu *sync.Mutex, op *[]string, mThread bool) {

	/// Trying to establish TCP connection
	conn, tcpErr := net.DialTimeout("tcp", socket, time.Second)

	/// If there is no connection err, then TCP port is opened
	if tcpErr == nil {
		defer conn.Close()
		print_opened(port, "tcp")

		/// Add to the list of opened ports using mutex
		mu.Lock()
		*op = append(*op, strconv.Itoa(port)+" (TCP)")
		mu.Unlock()

	} else {
		print_closed(port, "tcp")
		return
	}
}

func scan_udp_port(socket string, port int, mu *sync.Mutex, op *[]string, mThread bool) {

	laddr, _ := net.ResolveUDPAddr("udp4", "0.0.0.0")
	conn, err := net.ListenUDP("udp", laddr)

	if err != nil {
		return
	}
	defer conn.Close()

	dst, err := net.ResolveUDPAddr("udp4", socket)
	if err != nil {
		return
	}

	/// To check if the UDP port is open or not, we should receive a response from the port.
	_, err = conn.WriteTo([]byte("some message"), dst)
	if err != nil {
		return
	}

	/// Set deadline for reply in 0.5 sec
	conn.SetReadDeadline(time.Now().Add(time.Second / 2))
	readBuff := make([]byte, 1500)

	n, _, err := conn.ReadFrom(readBuff)

	/// If we get an error, then this UDP port is closed
	if err != nil {
		print_closed(port, "udp")
		return

	} else {
		_, err := icmp.ParseMessage(1, readBuff[:n])
		if err != nil {
			return
		}

		print_opened(port, "udp")

		mu.Lock()
		*op = append(*op, strconv.Itoa(port)+" (UDP)")
		mu.Unlock()
	}
}

func start_worker(portChan chan int, wg *sync.WaitGroup,
	mu *sync.Mutex, op *[]string, ip string) {

	defer wg.Done()

	for port := range portChan {
		socket := fmt.Sprintf("%s:%d", ip, port)
		scan_tcp_port(socket, port, mu, op, true)
		scan_udp_port(socket, port, mu, op, true)
	}
}

func start_scan(ip net.IP, ports []string, mThread bool,
	wg *sync.WaitGroup, mu *sync.Mutex, op *[]string, portsChan chan int) {

	for _, port := range ports {

		intPort, _ := strconv.Atoi(port)
		socket := fmt.Sprintf("%s:%d", ip, intPort)

		/// If we use a multithreading mode, we should send ports into channel for goroutines
		if mThread {
			portsChan <- intPort
		} else {
			scan_tcp_port(socket, intPort, mu, op, false)
			scan_udp_port(socket, intPort, mu, op, false)
		}
	}
}

func main() {

	fmt.Printf("Starting Gomap Tool at: %s\n", time.Now())

	ports, ip, mThread := init_parse()
	check_availability(ip)

	var openedPorts []string
	op := &openedPorts

	/// Start a pool of workers (goroutines) with port channel
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	portChan := make(chan int)

	if mThread {
		goroutines := runtime.NumCPU()
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go start_worker(portChan, wg, mu, op, ip.String())
		}
	}

	fmt.Println("Starting port scan:")
	fmt.Print("+-----------------------+----------------+\r\n", "|       Socket   	|",
		"     State   	 |\r\n", "+-----------------------+----------------+\r\n")

	start_time := time.Now()
	start_scan(ip, ports, mThread, wg, mu, op, portChan)

	close(portChan)

	/// Wait for all goroutines to finish
	wg.Wait()

	fmt.Println("+-----------------------+----------------+")
	fmt.Printf("Port scanning was finished in %s\n", time.Since(start_time))

	if len(openedPorts) > 0 {
		fmt.Println("List of opened ports: ", strings.Join(openedPorts, ", "))
	} else {
		fmt.Printf("There are no open TCP/UDP ports on target: %s", ip.String())
	}
}
