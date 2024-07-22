package wsnet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

type WSNConnect struct {
	Protocol string
	IP       string
	Port     uint16
	Bind     bool
	BindType uint8
}

func NewWSNConnect(protocol, ip string, port uint16, bind bool, bindtype uint8) *WSNConnect {
	return &WSNConnect{
		Protocol: protocol,
		IP:       ip,
		Port:     port,
		Bind:     bind,
		BindType: bindtype,
	}
}

func WSNConnectFromBytes(data []byte) (*WSNConnect, error) {
	return WSNConnectFromBuffer(bytes.NewReader(data))
}

func WSNConnectFromBuffer(buff *bytes.Reader) (*WSNConnect, error) {
	protocolBytes := make([]byte, 3)
	if _, err := buff.Read(protocolBytes); err != nil {
		return nil, err
	}
	protocol := string(protocolBytes)

	bindByte, err := buff.ReadByte()
	if err != nil {
		return nil, err
	}
	bind := bindByte == 1

	ipver, err := buff.ReadByte()
	if err != nil {
		return nil, err
	}

	var ip string
	if ipver == 0x04 {
		ipBytes := make([]byte, 4)
		if _, err := buff.Read(ipBytes); err != nil {
			return nil, err
		}
		ip = net.IP(ipBytes).String()
	} else if ipver == 0x06 {
		ipBytes := make([]byte, 16)
		if _, err := buff.Read(ipBytes); err != nil {
			return nil, err
		}
		ip = net.IP(ipBytes).String()
	} else if ipver == 0xFF {
		iplenBytes := make([]byte, 4)
		if _, err := buff.Read(iplenBytes); err != nil {
			return nil, err
		}
		iplen := binary.BigEndian.Uint32(iplenBytes)
		ipBytes := make([]byte, iplen)
		if _, err := buff.Read(ipBytes); err != nil {
			return nil, err
		}
		ip = string(ipBytes)
	}

	portBytes := make([]byte, 2)
	if _, err := buff.Read(portBytes); err != nil {
		return nil, err
	}
	port := binary.BigEndian.Uint16(portBytes)

	bindtypeByte, err := buff.ReadByte()
	if err != nil {
		return nil, err
	}
	bindtype := uint8(bindtypeByte)

	return NewWSNConnect(protocol, ip, port, bind, bindtype), nil
}

func (w *WSNConnect) ToData() ([]byte, error) {
	buff := new(bytes.Buffer)
	if _, err := buff.WriteString(w.Protocol); err != nil {
		return nil, err
	}

	if err := buff.WriteByte(BoolToByte(w.Bind)); err != nil {
		return nil, err
	}

	ip := net.ParseIP(w.IP)
	if ip == nil {
		ipLen := uint32(len(w.IP))
		buff.WriteByte(0xFF)
		binary.Write(buff, binary.BigEndian, ipLen)
		if _, err := buff.WriteString(w.IP); err != nil {
			return nil, err
		}
	} else {
		if ip.To4() != nil {
			buff.WriteByte(0x04)
			if _, err := buff.Write(ip.To4()); err != nil {
				return nil, err
			}
		} else if ip.To16() != nil {
			buff.WriteByte(0x06)
			if _, err := buff.Write(ip.To16()); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("invalid IP version")
		}
	}

	binary.Write(buff, binary.BigEndian, w.Port)
	buff.WriteByte(w.BindType)

	return buff.Bytes(), nil
}


