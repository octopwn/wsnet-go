package wsnet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

type WSNGetInfoReply struct {
	pid           string
	username      string // utf-16-le
	domain        string // utf-16-le
	logonserver   string // utf-16-le
	cpuarch       string
	hostname      string // utf-16-le
	usersid       string
	os            string
	logonserverip string
}

func normalizePlatform(goos string) string {
	switch goos {
	case "windows":
		return "WINDOWS"
	case "linux":
		return "LINUX"
	case "darwin":
		return "MACOS"
	default:
		// fallback for other GOOS values (freebsd, netbsd, etc.)
		return strings.ToUpper(goos)
	}
}

func NewWSNGetInfoReply(pid, username, domain, logonserver, cpuarch, hostname, usersid, os, logonserverip string) *WSNGetInfoReply {
	return &WSNGetInfoReply{
		pid:           pid,
		username:      username,
		domain:        domain,
		logonserver:   logonserver,
		cpuarch:       cpuarch,
		hostname:      hostname,
		usersid:       usersid,
		os:            os,
		logonserverip: logonserverip,
	}
}

// encodeUTF16LE takes a Go string s and returns a UTF-16-LE-encoded []byte.
func encodeUTF16LE(s string) []byte {
	// Convert s into a slice of UTF-16 code units
	u16 := utf16.Encode([]rune(s))

	// Prepare a buffer to hold the UTF-16-LE bytes
	buf := new(bytes.Buffer)

	// For each code unit, write it as a little-endian uint16
	for _, codeUnit := range u16 {
		// We use binary.Write with binary.LittleEndian
		// to write the 2 bytes of each UTF-16 code unit
		_ = binary.Write(buf, binary.LittleEndian, codeUnit)
	}

	return buf.Bytes()
}

func (w *WSNGetInfoReply) ToData() ([]byte, error) {
	buff := new(bytes.Buffer)

	// Write the pid
	fmt.Println("pid:", w.pid)
	pidLength := uint32(len(w.pid))
	binary.Write(buff, binary.BigEndian, pidLength)
	if _, err := buff.WriteString(w.pid); err != nil {
		return nil, err
	}

	// Write the username
	usernameBytes := encodeUTF16LE(w.username)
	usernameLength := uint32(len(usernameBytes))
	binary.Write(buff, binary.BigEndian, usernameLength)
	binary.Write(buff, binary.BigEndian, usernameBytes)

	// Write the domain
	domainBytes := encodeUTF16LE(w.domain)
	domainLength := uint32(len(domainBytes))
	binary.Write(buff, binary.BigEndian, domainLength)
	binary.Write(buff, binary.BigEndian, domainBytes)

	// Write the logonserver
	logonserverBytes := encodeUTF16LE(w.logonserver)
	logonserverLength := uint32(len(logonserverBytes))
	binary.Write(buff, binary.BigEndian, logonserverLength)
	binary.Write(buff, binary.BigEndian, logonserverBytes)

	// Write the cpuarch
	cpuarchLength := uint32(len(w.cpuarch))
	binary.Write(buff, binary.BigEndian, cpuarchLength)
	if _, err := buff.WriteString(w.cpuarch); err != nil {
		return nil, err
	}

	// Write the hostname
	hostnameBytes := encodeUTF16LE(w.hostname)
	hostnameLength := uint32(len(hostnameBytes))
	binary.Write(buff, binary.BigEndian, hostnameLength)
	binary.Write(buff, binary.BigEndian, hostnameBytes)

	// Write the usersid
	usersidLength := uint32(len(w.usersid))
	binary.Write(buff, binary.BigEndian, usersidLength)
	if _, err := buff.WriteString(w.usersid); err != nil {
		return nil, err
	}

	// Write the os
	osLength := uint32(len(w.os))
	binary.Write(buff, binary.BigEndian, osLength)
	if _, err := buff.WriteString(w.os); err != nil {
		return nil, err
	}

	// Write the logonserverip
	logonserveripLength := uint32(len(w.logonserverip))
	binary.Write(buff, binary.BigEndian, logonserveripLength)
	if _, err := buff.WriteString(w.logonserverip); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

type WSNErr struct {
	Reason string
	Extra  string
}

func NewWSNErr(reason, extra string) *WSNErr {
	return &WSNErr{
		Reason: reason,
		Extra:  extra,
	}
}

func WSNErrFromBytes(data []byte) (*WSNErr, error) {
	return WSNErrFromBuffer(bytes.NewReader(data))
}

func WSNErrFromBuffer(buff *bytes.Reader) (*WSNErr, error) {
	reasonLengthBytes := make([]byte, 4)
	if _, err := buff.Read(reasonLengthBytes); err != nil {
		return nil, err
	}
	reasonLength := binary.BigEndian.Uint32(reasonLengthBytes)

	reasonBytes := make([]byte, reasonLength)
	if _, err := buff.Read(reasonBytes); err != nil {
		return nil, err
	}
	reason := string(reasonBytes)

	extraLengthBytes := make([]byte, 4)
	if _, err := buff.Read(extraLengthBytes); err != nil {
		return nil, err
	}
	extraLength := binary.BigEndian.Uint32(extraLengthBytes)

	extraBytes := make([]byte, extraLength)
	if _, err := buff.Read(extraBytes); err != nil {
		return nil, err
	}
	extra := string(extraBytes)

	return NewWSNErr(reason, extra), nil
}

func (w *WSNErr) ToData() ([]byte, error) {
	buff := new(bytes.Buffer)

	// Write the reason
	reasonLength := uint32(len(w.Reason))
	binary.Write(buff, binary.BigEndian, reasonLength)
	if _, err := buff.WriteString(w.Reason); err != nil {
		return nil, err
	}

	// Write the extra
	extraLength := uint32(len(w.Extra))
	binary.Write(buff, binary.BigEndian, extraLength)
	if _, err := buff.WriteString(w.Extra); err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}
