package server

import (
	"context"
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
				dnsLayerPacket := rawPacket.Layer(layers.LayerTypeDNS)

				if dnsLayerPacket != nil {
					var dnsPacket = dnsLayerPacket.(*layers.DNS)
					go serveDNSPacket(handler, dnsPacket, cache)
				}
			}
		}
	}
}

func serveDNSPacket(handler *ConfigHandler, dnsPacket *layers.DNS, cache *Cache) {

	var config *ConfigInstance = handler.Get()
	udpExternal, errExt := net.ResolveUDPAddr("udp", config.Nameserver+":"+strconv.Itoa(config.ExternalPort))

	if errExt != nil {
		fmt.Println("\033[31mUnable to parse udp address\033[0m")
		os.Exit(0)
	}

	conn, err := net.DialUDP("udp", nil, udpExternal)
	if err != nil {
		fmt.Printf("\033[31mCan't start listening on port, %d\033[0m", config.ExternalPort)
		os.Exit(0)
	}
	defer conn.Close()

	if !dnsPacket.QR {
		for _, quest := range dnsPacket.Questions {
			if quest.Type.String() == "A" {
				conn.WriteToUDP(dnsPacket.Contents, udpExternal)
				fmt.Print("Writed")

				fmt.Println(len(dnsPacket.Contents))
			}
		}
	}
}
