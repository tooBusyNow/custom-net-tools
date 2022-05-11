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
	serveRequest(handler, mainContext, conn, cache)
}

func serveRequest(handler *ConfigHandler, mainContext context.Context, conn *net.UDPConn, cache *Cache) {
	var buffer [1024]byte
	for {
		select {
		case <-mainContext.Done():
			conn.Close()
			return

		default:
			if handler.NeedRestart {
				conn.Close()
				return
			}
			conn.SetReadDeadline(time.Now().Add(time.Second))
			_, addr, _ := conn.ReadFromUDP(buffer[:])

			if addr != nil {
				rawPacket := gopacket.NewPacket(buffer[:], layers.LayerTypeDNS, gopacket.Default)
				dnsPacket := rawPacket.Layer(layers.LayerTypeDNS).(*layers.DNS)

				if dnsPacket != nil {
					go serveDNSPacket(handler, handler.Get().Nameserver, dnsPacket, cache)
				}
			}
		}
	}
}

func serveDNSPacket(handler *ConfigHandler, serverIP string, dnsPacket *layers.DNS, cache *Cache) error {

	var config *ConfigInstance = handler.Get()
	var someErr error

	udpExternal := serverIP + ":" + strconv.Itoa(config.ExternalPort)
	conn, err := net.Dial("udp", udpExternal)
	if err != nil {
		fmt.Println("\033[31mUnable to establish connection to External DNS server\033[0m", udpExternal)
		os.Exit(0)
	}

	fmt.Println(serverIP)

	for _, quest := range dnsPacket.Questions {
		if quest.Type.String() == "A" {
			p := make([]byte, 2048)
			fmt.Fprintf(conn, string(dnsPacket.Contents))

			conn.SetReadDeadline(time.Now().Add(time.Second * 3))
			_, err = bufio.NewReader(conn).Read(p)

			if err != nil {
				return errors.New("Timeout")
			}
			conn.Close()

			rawPacket := gopacket.NewPacket(p[:], layers.LayerTypeDNS, gopacket.Default)
			dnsResponse := rawPacket.Layer(layers.LayerTypeDNS).(*layers.DNS)

			fmt.Println(dnsResponse.ResponseCode)

			if len(dnsResponse.Answers) > 0 {
				fmt.Println("Finished")
				fmt.Println(dnsResponse.Answers[0].IP)
				return nil
			} else if dnsResponse.ResponseCode.String() == "Format Error" {
				return errors.New("FE")
			} else {
				for _, nsIP := range dnsResponse.Additionals {
					fmt.Println("Got nameserver: ", nsIP.IP.String())

					if nsIP.IP.To4() == nil {
						continue
					}

					someErr = serveDNSPacket(handler, nsIP.IP.String(), dnsPacket, cache)
					fmt.Println(someErr)
					if someErr == nil {
						break
					}
				}
			}

		}
	}
	return someErr
}
