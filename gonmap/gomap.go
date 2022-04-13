package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func ports_parse(portRange string) []string {

	reg := regexp.MustCompile(`[-,]`)
	scanRange := reg.Split(portRange, -1)

	if len(scanRange) == 0 || len(scanRange) > 2 {
		fmt.Println("Scan range should be either single number, or two numbers separated with \"-\"")
		os.Exit(0)
	}

	for _, raw_port := range scanRange {
		if _, err := strconv.Atoi(raw_port); err != nil {
			fmt.Println("Can't parse port numbers")
			os.Exit(0)
		}
	}
	return scanRange
}

func init_parse() (ports []string, ip net.IP, mTh bool) {

	var portRange string
	flag.StringVar(&portRange, "p", "80", "specifies port range to scan")

	var mThread bool
	flag.BoolVar(&mThread, "t", false, "allows to run with goroutines (parallel)")
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

func check_avail(ip net.IP) {

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
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
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

	conn, tcpErr := net.DialTimeout("tcp", socket, time.Second)

	if tcpErr == nil {
		defer conn.Close()
		fmt.Printf("|%d/tcp ⟸\t\t| %s	 |\n", port, "Opened ✔")
		mu.Lock()

	} else {
		fmt.Printf("|%d/tcp\t\t| %s	 |\n", port, "Closed ✖")
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

	_, err = conn.WriteTo([]byte("some message"), dst)
	if err != nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(time.Second / 2))
	readBuff := make([]byte, 1500)

	n, _, err := conn.ReadFrom(readBuff)
	if err != nil {
		fmt.Printf("|%d/udp\t\t| %s	 |\n", port, "Closed ✖")
		return
	} else {
		_, err := icmp.ParseMessage(1, readBuff[:n])
		if err != nil {
			return
		}
		fmt.Printf("|%d/udp ⟸\t\t| %s	 |\n", port, "Opened ✔")
	}
}

func start_worker(portsChan chan int, wg *sync.WaitGroup,
	mu *sync.Mutex, op *[]string, ip string) {

	defer wg.Done()

	for port := range portsChan {
		socket := fmt.Sprintf("%s:%d", ip, port)
		scan_tcp_port(socket, port, mu, op, true)
		scan_udp_port(socket, port, mu, op, true)
	}
}

func start_scan(ip net.IP, ports []string, mThread bool,
	wg *sync.WaitGroup, mu *sync.Mutex, op *[]string, portsChan chan int) {

	startPort, _ := strconv.Atoi(ports[0])
	finishPort, _ := strconv.Atoi(ports[0])

	if len(ports) > 1 {
		finishPort, _ = strconv.Atoi(ports[1])
	}

	fmt.Println("Starting port scan:")
	fmt.Println("+-----------------------+----------------+\r\n|       Socket   	|",
		"    State   	 |\n+-----------------------+----------------+")

	for port := startPort; port <= finishPort; port++ {
		socket := fmt.Sprintf("%s:%d", ip, port)

		if mThread {
			portsChan <- port
		} else {
			scan_tcp_port(socket, port, mu, op, false)
			scan_udp_port(socket, port, mu, op, false)
		}
	}
}

func main() {

	start_time := time.Now()
	fmt.Printf("Starting Gomap Tool at: %s\n", start_time.String())

	ports, ip, mThread := init_parse()

	check_avail(ip)

	var openedPorts []string
	op := &openedPorts

	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	portsChan := make(chan int)

	/// Actual number of goroutines will be NumCPU x 2.
	/// StartWorker runs in a separate goroutine just
	/// to start UPD and TCP goroutines

	goroutines := runtime.NumCPU()
	if mThread {
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go start_worker(portsChan, wg, mu, op, ip.String())
		}
	}

	start_scan(ip, ports, mThread, wg, mu, op, portsChan)
	close(portsChan)

	wg.Wait()

	fmt.Println("+-----------------------+----------------+")
	fmt.Printf("Port scanning was finished in %s", time.Since(start_time).String())
}
