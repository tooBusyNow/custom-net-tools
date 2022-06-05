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
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var rootIP string

func StartServer(handler *ConfigHandler, mainContext context.Context, cache *Cache) {
	var config *ConfigInstance = handler.Get()

	var udpAddr = &net.UDPAddr{
		IP:   net.ParseIP(config.Host),
		Port: 53,
	}
	conn, err := net.ListenUDP("udp", udpAddr)

	if err != nil {
		fmt.Printf("\033[31mCan't start listening on port, %d\033[0m", 53)
		fmt.Println(err)
		os.Exit(0)
	}

	rootIP = handler.Get().Nameserver
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
			intConn.SetReadDeadline(time.Now().Add(time.Second * 1))
			n, intAddr, _ := intConn.ReadFromUDP(buffer[:])
			data := buffer[:]

			if intAddr != nil {
				newPacket := gopacket.NewPacket(buffer[:n], layers.LayerTypeDNS, gopacket.Default)
				rawPacket := newPacket.Layer(layers.LayerTypeDNS)

				if rawPacket != nil {
					dnsInternalReq := rawPacket.(*layers.DNS)
					if dnsInternalReq != nil {
						go serveDNSPacket(handler, rootIP, *dnsInternalReq, cache, intConn, intAddr, data)
					}
				}
			}
		}
	}
}

func serveDNSPacket(handler *ConfigHandler, dstServerIP string, dnsIntReq layers.DNS,
	cache *Cache, intConn *net.UDPConn, intAddr *net.UDPAddr, data []byte) {

	var bytes []byte
	var AAFlag bool
	var NoResFlag bool
	var dnsResponse layers.DNS
	var initialReq layers.DNS = dnsIntReq
	var depth int = 0

	for {

		if depth > 8 {
			return
		}
		bytes, AAFlag, dnsResponse, NoResFlag = sendDNSRequest(dstServerIP, dnsIntReq, AAFlag, NoResFlag)

		if dnsResponse.ResponseCode.String() == "Server Failure " {
			dstServerIP = handler.Get().Nameserver
			NoResFlag = true
			continue
		}

		if AAFlag {
			if string(dnsIntReq.Questions[0].Name) == string(initialReq.Questions[0].Name) {
				intConn.WriteTo(getSerializedDNSPacket(dnsResponse), intAddr)
				return
			} else {
				newReq := getATypeReqForName(dnsIntReq, string(initialReq.Questions[0].Name))
				dnsIntReq = newReq
			}
		}

		rawAddr := net.ParseIP(string(bytes))
		if rawAddr != nil {
			dstServerIP = rawAddr.String()
		} else {
			newReq := getATypeReqForName(dnsIntReq, string(bytes))
			dnsIntReq = newReq
			dstServerIP = handler.Get().Nameserver
		}

		depth += 1
	}
}

func sendDNSRequest(dstServerIP string, dnsIntReq layers.DNS, AAFlag bool, NoResFlag bool) ([]byte, bool, layers.DNS, bool) {

	for _, quest := range dnsIntReq.Questions {

		switch quest.Type {
		case layers.DNSTypeA:
			dnsResponse := resendToExternalWait4Response(dstServerIP, dnsIntReq)
			if dnsResponse.AA {
				AAFlag = true
			}

			if dnsResponse.ResponseCode == 2 && !NoResFlag {
				return dnsIntReq.Contents, AAFlag, dnsResponse, NoResFlag
			}

			if len(dnsResponse.Additionals) > 0 {
				for _, addRecord := range dnsResponse.Additionals {
					if addRecord.IP.To4() == nil {
						continue
					}
					if NoResFlag {
						NoResFlag = false
						continue
					}
					NoResFlag = false
					dstServerIP = addRecord.IP.String()
					return []byte(dstServerIP), AAFlag, dnsResponse, NoResFlag
				}

			} else if len(dnsResponse.Authorities) > 0 {
				return dnsResponse.Authorities[0].NS, AAFlag, dnsResponse, NoResFlag

			} else if len(dnsResponse.Answers) > 0 {
				return []byte(dnsResponse.Answers[0].IP.String()), AAFlag, dnsResponse, NoResFlag
			}

		case layers.DNSTypePTR:
			if string(quest.Name) == "1.0.0.127.in-addr.arpa" {
				replyMess := getPTRecord4LocalResolver(dnsIntReq)
				bytes := getSerializedDNSPacket(replyMess)
				return bytes, AAFlag, replyMess, NoResFlag
			}
		}
	}
	return []byte(""), AAFlag, layers.DNS{}, NoResFlag
}

func getPTRecord4LocalResolver(dnsIntReq layers.DNS) layers.DNS {
	var dnsAnswer layers.DNSResourceRecord = layers.DNSResourceRecord{
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,

		Name: []byte(dnsIntReq.Questions[0].Name),
		PTR:  []byte("GoDNSResolver"),
		TTL:  90,
	}

	var replyMess layers.DNS = layers.DNS{
		ID: dnsIntReq.ID,

		QR: true,
		RD: true,
		AA: false,
		TC: false,

		QDCount: 1,
		ANCount: 1,

		OpCode:    layers.DNSOpCodeQuery,
		Questions: dnsIntReq.Questions,
		Answers:   append(dnsIntReq.Answers, dnsAnswer),
	}

	return replyMess
}

func getATypeReqForName(dnsIntReq layers.DNS, newName string) layers.DNS {

	newDNSReq := dnsIntReq

	var dnsQuest layers.DNSQuestion
	dnsQuest.Class = layers.DNSClassIN
	dnsQuest.Type = layers.DNSTypeA
	dnsQuest.Name = []byte(newName)

	newDNSReq.Questions = []layers.DNSQuestion{dnsQuest}

	test := getSerializedDNSPacket(newDNSReq)
	decoded := gopacket.NewPacket([]byte(test), layers.LayerTypeDNS, gopacket.Default)
	test2 := decoded.Layer(layers.LayerTypeDNS).(*layers.DNS)

	test3 := *test2

	return test3
}

func getSerializedDNSPacket(replyMess layers.DNS) []byte {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}
	err := replyMess.SerializeTo(buf, opts)

	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func openExternalConn(dstServerIP string) *net.UDPConn {
	udpExternal := &net.UDPAddr{
		Port: 53,
		IP:   net.ParseIP(dstServerIP),
	}
	extConn, err := net.DialUDP("udp", nil, udpExternal)
	if err != nil {
		fmt.Println("\033[31mUnable to establish connection to External DNS server\033[0m", udpExternal)
		os.Exit(0)
	}
	return extConn
}

func resendToExternalWait4Response(dstServerIP string, dnsIntReq layers.DNS) layers.DNS {

	extConn := openExternalConn(dstServerIP)
	defer extConn.Close()

	p := make([]byte, 2048)
	extConn.Write(dnsIntReq.Contents)

	extConn.SetReadDeadline(time.Now().Add(time.Second / 2))
	n, err := bufio.NewReader(extConn).Read(p)

	if err != nil {
		fmt.Println(errors.New("Timeout"))
	}

	rawPacket := gopacket.NewPacket(p[:n], layers.LayerTypeDNS, gopacket.Default)
	dnsResponse := rawPacket.Layer(layers.LayerTypeDNS).(*layers.DNS)

	return *dnsResponse
}
