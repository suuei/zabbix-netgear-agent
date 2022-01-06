package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

func ParseAll(obj map[string]interface{}, buf []byte) {
	for p := 0; p < len(buf); {
		dataType := binary.BigEndian.Uint16(buf[p : p+2])
		p += 2
		dataLength := binary.BigEndian.Uint16(buf[p : p+2])
		p += 2

		if DEBUG {
			fmt.Fprintf(os.Stderr, "===== parse =====\n")
			fmt.Fprintf(os.Stderr, "Type: %#04x, Length: %d\n", dataType, dataLength)
		}

		dataBody := buf[p : p+int(dataLength)]
		p += int(dataLength)

		switch dataType {
		case 0x0001:
			ParseModel(obj, dataBody)
		case 0x0003:
			ParseHostname(obj, dataBody)
		case 0x0004:
			ParseMacAddress(obj, dataBody)
		case 0x0005:
			ParseLocation(obj, dataBody)
		case 0x0006:
			ParseIPAddress(obj, dataBody)

		// interfaces
		case 0x0c00:
			ParseSpeedStat(obj, dataBody)
		case 0x1000:
			ParsePortStat(obj, dataBody)

		case 0xFFFF:
			return
		}
	}
}

// 0x0001
func ParseModel(obj map[string]interface{}, buf []byte) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, "Model=%v\n", string(buf))
	}

	obj["model"] = string(buf)
}

// 0x0003
func ParseHostname(obj map[string]interface{}, buf []byte) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, "Hostname=%v\n", string(buf))
	}

	obj["hostname"] = string(buf)
}

// 0x0004
func ParseMacAddress(obj map[string]interface{}, buf []byte) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, "MAC=%02X:%02X:%02X:%02X:%02X:%02X\n",
			buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	}

	obj["macaddress"] = string(fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]))
}

// 0x0005
func ParseLocation(obj map[string]interface{}, buf []byte) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, "Location=%v\n", string(buf))
	}

	obj["location"] = string(buf)
}

// 0x0006
func ParseIPAddress(obj map[string]interface{}, buf []byte) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, "IP=%d.%d.%d.%d\n",
			buf[0], buf[1], buf[2], buf[3])
	}

	obj["ipaddress"] = string(fmt.Sprintf("%d.%d.%d.%d",
		buf[0], buf[1], buf[2], buf[3]))
}

// 0x0c00
func ParseSpeedStat(obj map[string]interface{}, buf []byte) {
	if DEBUG {
		fmt.Fprintf(os.Stderr, "Port #%d = %d\n", buf[0], buf[1])
	}

	keyname := fmt.Sprintf("interface%d", buf[0])
	if obj[keyname] == nil {
		obj[keyname] = map[string]interface{}{}
	}

	if interfaceObj, ok := obj[keyname].(map[string]interface{}); ok {
		interfaceObj["speed"] = buf[1]
	}
}

// 0x1000
type NetgearPacketPortStat struct {
	PortNo           uint8
	RecvBytes        uint64
	SentBytes        uint64
	Packets          uint64
	BroadcastPackets uint64
	MulticastPackets uint64
	CRCErrors        uint64
}

// 0x1000
func ParsePortStat(obj map[string]interface{}, buf []byte) {
	reader := bytes.NewReader(buf)
	portStat := new(NetgearPacketPortStat)
	binary.Read(reader, binary.BigEndian, portStat)

	if DEBUG {
		fmt.Fprintf(os.Stderr, "Port #%d\n", portStat.PortNo)
		fmt.Fprintf(os.Stderr, "  RecvBytes: %15d\n", portStat.RecvBytes)
		fmt.Fprintf(os.Stderr, "  SentBytes: %15d\n", portStat.SentBytes)
		fmt.Fprintf(os.Stderr, "  CRC Error: %15d\n", portStat.CRCErrors)
	}

	keyname := fmt.Sprintf("interface%d", buf[0])
	if obj[keyname] == nil {
		obj[keyname] = map[string]interface{}{}
	}

	if interfaceObj, ok := obj[keyname].(map[string]interface{}); ok {
		interfaceObj["recv"] = portStat.RecvBytes
		interfaceObj["sent"] = portStat.SentBytes
		interfaceObj["error"] = portStat.CRCErrors
	}
}
