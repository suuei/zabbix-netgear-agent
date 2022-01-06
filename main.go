package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

var DEBUG = false

type NetgearPacketHeader struct {
	PacketType uint16
	IsError    uint16
	ErrorCode  uint16
	Reserved1  [2]byte
	SrcMacAddr [6]byte
	DstMacAddr [6]byte
	Reserved2  [2]byte
	Sequence   uint16
	Magic      [8]byte
}

func NewNetgearPacketHeader() *NetgearPacketHeader {
	ret := new(NetgearPacketHeader)
	ret.PacketType = 0x0101
	ret.Magic = [8]byte{'N', 'S', 'D', 'P', 0, 0, 0, 0}
	return ret
}

func DumpHex(buf []byte) {
	for i := 0; i < len(buf); i++ {
		fmt.Fprintf(os.Stderr, "%02X", buf[i])

		if i%16 == 15 {
			fmt.Fprint(os.Stderr, "\n")
		} else if i%4 == 3 {
			fmt.Fprint(os.Stderr, "  ")
		} else {
			fmt.Fprint(os.Stderr, " ")
		}
	}
}

func SetMacAddressToHeader(header *NetgearPacketHeader) {
	// fault tolerant
	header.SrcMacAddr[5] = 0x01

	iflist, err := net.Interfaces()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s\n", err.Error())
		os.Exit(1)
	}
	for _, ifobj := range iflist {
		addr, err := ifobj.Addrs()
		if err != nil {
			continue
		}
		if len(ifobj.HardwareAddr) == 6 && len(addr) > 0 {
			for i := 0; i < 6; i++ {
				header.SrcMacAddr[i] = ifobj.HardwareAddr[i]
			}
			break
		}
	}
}

func CreateConn(host string) (conn *net.UDPConn, err error) {
	udpAddrRecv := &net.UDPAddr{
		Port: 63321,
	}

	udpAddrTarget, err := net.ResolveUDPAddr("udp4", host+":63322")
	if err != nil {
		return nil, err
	}

	conn, err = net.DialUDP("udp4", udpAddrRecv, udpAddrTarget)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func GetDeviceData(host string) (obj map[string]interface{}, status int) {
	conn, err := CreateConn(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s\n", err.Error())
		os.Exit(1)
	}
	rand.Seed(time.Now().UnixNano())

	requestHeader := NewNetgearPacketHeader()
	requestHeader.Sequence = uint16(rand.Uint32())
	SetMacAddressToHeader(requestHeader)

	var requestBuffer bytes.Buffer
	binary.Write(&requestBuffer, binary.BigEndian, requestHeader)
	requestBuffer.Write([]byte{0x00, 0x01, 0x00, 0x00}) // MODEL
	requestBuffer.Write([]byte{0x00, 0x03, 0x00, 0x00}) // NAME
	requestBuffer.Write([]byte{0x00, 0x04, 0x00, 0x00}) // MACADDRESS
	requestBuffer.Write([]byte{0x00, 0x05, 0x00, 0x00}) // LOCATION
	requestBuffer.Write([]byte{0x00, 0x06, 0x00, 0x00}) // IPADDRESS
	requestBuffer.Write([]byte{0x0c, 0x00, 0x00, 0x00}) // SPEED_STAT
	requestBuffer.Write([]byte{0x10, 0x00, 0x00, 0x00}) // PORT_STAT
	requestBuffer.Write([]byte{0xff, 0xff, 0x00, 0x00}) // END

	_, err = conn.Write(requestBuffer.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s\n", err.Error())
		os.Exit(1)
	}

	var responseBuf [1024]byte
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(responseBuf[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s\n", err.Error())
		return nil, -1
	}

	responseReader := bytes.NewReader(responseBuf[:])
	responseHeader := new(NetgearPacketHeader)
	binary.Read(responseReader, binary.BigEndian, responseHeader)

	ret := map[string]interface{}{}

	if responseHeader.IsError != 0 {
		fmt.Fprintf(os.Stderr, "Error code: %#4x\n", responseHeader.ErrorCode)
		ret["status"] = 1
		return ret, 1
	}

	if DEBUG {
		fmt.Fprintf(os.Stderr, "===== body =====\n")
		DumpHex(responseBuf[32:n])
	}

	ParseAll(ret, responseBuf[32:n])
	ret["status"] = 0

	return ret, 0
}

func DiscoveryDevices(host string) (obj map[string]interface{}, status int) {
	ret := map[string]interface{}{"message": "Not Implemented yet."}

	return ret, 1
}

func main() {
	var (
		host    = flag.String("host", "localhost", "host address")
		mode    = flag.String("mode", "get", "Mode flag")
		isDebug = flag.Bool("debug", false, "Debug flag")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "mode:")
		fmt.Fprintln(os.Stderr, "  get = Get Device Data")
		fmt.Fprintln(os.Stderr, "  discoverif = Discover Device Interfaces")
		fmt.Fprintln(os.Stderr, "  discoverdev = Discover Devices (please set host to broadcast address)")
	}

	flag.Parse()

	DEBUG = *isDebug
	var obj map[string]interface{}
	var status int

	switch *mode {
	case "get":
		for i := 0; i < 3; i++ {
			obj, status = GetDeviceData(*host)
			if status == 0 {
				break
			}
		}

	case "discoverif":
		obj, status = GetDeviceData(*host)
		dataobj := []map[string]string{}
		for key := range obj {
			if strings.HasPrefix(key, "interface") {
				dataobj = append(dataobj, map[string]string{"{#PATH}": key, "{#PORT}": key[9:]})
			}
		}
		obj = map[string]interface{}{"data": dataobj}

	case "discoverdev":
		obj, status = DiscoveryDevices(*host)

	default:
		flag.Usage()
		os.Exit(1)
	}

	ret, err := json.Marshal(obj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println(string(ret))
	os.Exit(status)
}
