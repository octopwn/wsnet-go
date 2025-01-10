package wsnet

import (
    "bytes"
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    "net"
)

// WSNServerSocketData is the Go equivalent of your Python class,
// ignoring the "token" field entirely.
type WSNServerSocketData struct {
    ConnectionToken [16]byte
    IPVersion       byte   // Derived from reading the next 1 byte
    ClientIP        string
    ClientPort      uint16
    Data            []byte
}

func WSNClientSocketDataFromBytes(b []byte) (*WSNServerSocketData, error) {
    buf := bytes.NewReader(b)

    var obj WSNServerSocketData
    // Read 16 bytes for connectiontoken
    if _, err := io.ReadFull(buf, obj.ConnectionToken[:]); err != nil {
        return nil, fmt.Errorf("failed to read connectiontoken: %w", err)
    }

    // Read 1 byte for IP version
    if err := binary.Read(buf, binary.BigEndian, &obj.IPVersion); err != nil {
        return nil, fmt.Errorf("failed to read IP version: %w", err)
    }

    // Parse IP address based on IPVersion
    switch obj.IPVersion {
    case 0x04:
        // IPv4 -> read 4 bytes
        ipBytes := make([]byte, 4)
        if _, err := io.ReadFull(buf, ipBytes); err != nil {
            return nil, fmt.Errorf("failed to read IPv4: %w", err)
        }
        obj.ClientIP = net.IP(ipBytes).String()

    case 0x06:
        // IPv6 -> read 16 bytes
        ipBytes := make([]byte, 16)
        if _, err := io.ReadFull(buf, ipBytes); err != nil {
            return nil, fmt.Errorf("failed to read IPv6: %w", err)
        }
        obj.ClientIP = net.IP(ipBytes).String()

    case 0xFF:
        // "Arbitrary-length" IP => first 4 bytes = length, then read that many
        var ipLen uint32
        if err := binary.Read(buf, binary.BigEndian, &ipLen); err != nil {
            return nil, fmt.Errorf("failed to read ip length: %w", err)
        }
        ipBytes := make([]byte, ipLen)
        if _, err := io.ReadFull(buf, ipBytes); err != nil {
            return nil, fmt.Errorf("failed to read ip string: %w", err)
        }
        obj.ClientIP = string(ipBytes)

    default:
        return nil, fmt.Errorf("invalid IP version: 0x%X", obj.IPVersion)
    }

    // Read 2-byte port (big-endian)
    if err := binary.Read(buf, binary.BigEndian, &obj.ClientPort); err != nil {
        return nil, fmt.Errorf("failed to read client port: %w", err)
    }

    // The rest is data
    remaining, err := io.ReadAll(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read remaining data: %w", err)
    }
    obj.Data = remaining

    return &obj, nil
}


func (w *WSNServerSocketData) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // Write 16-byte connectiontoken
    if _, err := buf.Write(w.ConnectionToken[:]); err != nil {
        return nil, fmt.Errorf("failed to write connectiontoken: %w", err)
    }

    // Determine IP version byte from the client IP, or store w.IPVersion as-is.
    // If you want the Python-like logic, you'd do something like:
    ip := net.ParseIP(w.ClientIP)
    if ip == nil {
        // If it's not a valid IP, treat it like the Python "ipaddress.ip_address" exception
        // => fallback to 0xFF
        if err := buf.WriteByte(0xFF); err != nil {
            return nil, fmt.Errorf("failed to write 0xFF ipver: %w", err)
        }
        // Write length + raw string
        ipBytes := []byte(w.ClientIP)
        if err := binary.Write(&buf, binary.BigEndian, uint32(len(ipBytes))); err != nil {
            return nil, fmt.Errorf("failed to write ip length: %w", err)
        }
        if _, err := buf.Write(ipBytes); err != nil {
            return nil, fmt.Errorf("failed to write ip string: %w", err)
        }
    } else {
        // It's a valid IP
        if ip.To4() != nil {
            // IPv4
            if err := buf.WriteByte(0x04); err != nil {
                return nil, fmt.Errorf("failed to write 0x04 ipver: %w", err)
            }
            if _, err := buf.Write(ip.To4()); err != nil {
                return nil, fmt.Errorf("failed to write IPv4 bytes: %w", err)
            }
        } else if ip.To16() != nil {
            // IPv6
            if err := buf.WriteByte(0x06); err != nil {
                return nil, fmt.Errorf("failed to write 0x06 ipver: %w", err)
            }
            if _, err := buf.Write(ip.To16()); err != nil {
                return nil, fmt.Errorf("failed to write IPv6 bytes: %w", err)
            }
        } else {
            return nil, errors.New("unrecognized IP address format")
        }
    }

    // Write port as 2-byte big-endian
    if err := binary.Write(&buf, binary.BigEndian, w.ClientPort); err != nil {
        return nil, fmt.Errorf("failed to write client port: %w", err)
    }

    // Finally, write the rest of data
    if _, err := buf.Write(w.Data); err != nil {
        return nil, fmt.Errorf("failed to write data: %w", err)
    }

    return buf.Bytes(), nil
}