package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	. "godns/cache"
	. "godns/config"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func StartServer(handler *ConfigHandler, mainContext context.Context, cache *Cache) {
	var config *ConfigInstance = handler.Get()

	var udpAddr = &net.UDPAddr{
		IP:   net.ParseIP(config.Host),
		Port: config.InternalPort,
	}
	conn, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		fmt.Printf("\033[31mCan't start listening on port, %d\033[0m", config.InternalPort)
		os.Exit(0)
	}

	fmt.Print("\033[32mDNS Server is up and running\n\033[0m")
	go serveRequest(handler, mainContext, conn, cache)
}

func serveRequest(handler *ConfigHandler, mainContext context.Context, intConn *net.UDPConn, cache *Cache) {
	var buffer [1024]byte
	for {
		select {
		case <-mainContext.Done():
			intConn.Close()
			return

		default:
			if handler.NeedRestart {
				intConn.Close()
				return
			}
			intConn.SetReadDeadline(time.Now().Add(time.Second * 3))
			n, addr, _ := intConn.ReadFromUDP(buffer[:])
			if addr != nil {
				rawPacket := gopacket.NewPacket(buffer[:n], layers.LayerTypeDNS, gopacket.Default)
				dnsPacket := rawPacket.Layer(layers.LayerTypeDNS).(*layers.DNS)

				if dnsPacket != nil {
					go serveDNSPacket(handler, handler.Get().Nameserver, dnsPacket, cache, intConn)
				}
			}
		}
	}
}

func serveDNSPacket(handler *ConfigHandler, serverIP string,
	dnsPacket *layers.DNS, cache *Cache, intConn *net.UDPConn) error {

	var config *ConfigInstance = handler.Get()
	var someErr error

	udpExternal := serverIP + ":" + strconv.Itoa(config.ExternalPort)
	extConn, err := net.Dial("udp", udpExternal)
	if err != nil {
		fmt.Println("\033[31mUnable to establish connection to External DNS server\033[0m", udpExternal)
		os.Exit(0)
	}

	for _, quest := range dnsPacket.Questions {
		if quest.Type.String() == "A" {
			p := make([]byte, 2048)
			fmt.Fprintf(extConn, string(dnsPacket.Contents))

			extConn.SetReadDeadline(time.Now().Add(time.Second * 3))
			n, err := bufio.NewReader(extConn).Read(p)

			if err != nil {
				return errors.New("Timeout")
			}
			extConn.Close()
			rawPacket := gopacket.NewPacket(p[:n], layers.LayerTypeDNS, gopacket.Default)
			dnsResponse := rawPacket.Layer(layers.LayerTypeDNS).(*layers.DNS)

			if len(dnsResponse.Answers) > 0 {
				fmt.Println("Finished")
				fmt.Println(dnsResponse.Answers[0].IP)
				var udpAddr = &net.UDPAddr{
					IP:   net.ParseIP(config.Host),
					Port: config.InternalPort,
				}
				intConn.WriteToUDP(dnsResponse.Contents, udpAddr)

				return nil

			} else {
				for _, nsIP := range dnsResponse.Additionals {
					if nsIP.IP.To4() == nil {
						continue
					}
					someErr = serveDNSPacket(handler, nsIP.IP.String(), dnsPacket, cache, intConn)
					if someErr == nil {
						break
					}
				}
			}
		}
	}
	return someErr
}
