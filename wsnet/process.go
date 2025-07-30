package wsnet

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "time"
)

// WSNProcessInfo represents information about a single process on the remote system.
// The binary layout (big-endian) matches the Python implementation:
//  • uint64  PID
//  • string  Name
//  • string  WindowTitle
//  • string  BinPath
//  • uint64  MemoryUsage
//  • uint64  ThreadCount
//  • uint64  StartTime (unix seconds, 0 if unknown)
//  • uint64  CPUTime   (seconds, 0 if unknown)
//  • byte    IsResponding (0x00 / 0x01)
//  • string  MainWindowTitle
//
// NOTE: The surrounding packet (WSPacket) carries the command type and token, so
// they are intentionally **not** included here.
type WSNProcessInfo struct {
    PID             uint64
    Name            string
    WindowTitle     string
    BinPath         string
    MemoryUsage     uint64
    ThreadCount     uint64
    StartTime       time.Time      // zero value if unknown
    CPUTime         time.Duration  // zero value if unknown
    IsResponding    bool
    MainWindowTitle string
}

// WSNProcessInfoFromBytes parses a WSNProcessInfo structure from raw bytes.
func WSNProcessInfoFromBytes(data []byte) (*WSNProcessInfo, error) {
    return WSNProcessInfoFromBuffer(bytes.NewReader(data))
}

// WSNProcessInfoFromBuffer reads a WSNProcessInfo from the provided buffer.
func WSNProcessInfoFromBuffer(buf *bytes.Reader) (*WSNProcessInfo, error) {
    var info WSNProcessInfo

    // PID
    if err := binary.Read(buf, binary.BigEndian, &info.PID); err != nil {
        return nil, fmt.Errorf("failed to read pid: %w", err)
    }

    // Name
    name, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read name: %w", err)
    }
    info.Name = name

    // WindowTitle
    wt, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read window title: %w", err)
    }
    info.WindowTitle = wt

    // BinPath
    bp, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read binpath: %w", err)
    }
    info.BinPath = bp

    // MemoryUsage
    if err := binary.Read(buf, binary.BigEndian, &info.MemoryUsage); err != nil {
        return nil, fmt.Errorf("failed to read memory usage: %w", err)
    }

    // ThreadCount
    if err := binary.Read(buf, binary.BigEndian, &info.ThreadCount); err != nil {
        return nil, fmt.Errorf("failed to read thread count: %w", err)
    }

    // StartTime (unix seconds)
    var startUnix uint64
    if err := binary.Read(buf, binary.BigEndian, &startUnix); err != nil {
        return nil, fmt.Errorf("failed to read start time: %w", err)
    }
    if startUnix != 0 {
        info.StartTime = time.Unix(int64(startUnix), 0)
    }

    // CPUTime (seconds)
    var cpuSeconds uint64
    if err := binary.Read(buf, binary.BigEndian, &cpuSeconds); err != nil {
        return nil, fmt.Errorf("failed to read cpu time: %w", err)
    }
    if cpuSeconds != 0 {
        info.CPUTime = time.Duration(cpuSeconds) * time.Second
    }

    // IsResponding (1 byte)
    isRespByte, err := buf.ReadByte()
    if err != nil {
        return nil, fmt.Errorf("failed to read is_responding: %w", err)
    }
    info.IsResponding = isRespByte != 0

    // MainWindowTitle
    mwt, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read main window title: %w", err)
    }
    info.MainWindowTitle = mwt

    return &info, nil
}

// ToData serializes the WSNProcessInfo to the binary representation expected by the protocol.
func (w *WSNProcessInfo) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // PID
    if err := binary.Write(&buf, binary.BigEndian, w.PID); err != nil {
        return nil, fmt.Errorf("failed to write pid: %w", err)
    }

    // Name
    if err := writeStr(&buf, w.Name); err != nil {
        return nil, fmt.Errorf("failed to write name: %w", err)
    }

    // WindowTitle
    if err := writeStr(&buf, w.WindowTitle); err != nil {
        return nil, fmt.Errorf("failed to write window title: %w", err)
    }

    // BinPath
    if err := writeStr(&buf, w.BinPath); err != nil {
        return nil, fmt.Errorf("failed to write binpath: %w", err)
    }

    // MemoryUsage
    if err := binary.Write(&buf, binary.BigEndian, w.MemoryUsage); err != nil {
        return nil, fmt.Errorf("failed to write memory usage: %w", err)
    }

    // ThreadCount
    if err := binary.Write(&buf, binary.BigEndian, w.ThreadCount); err != nil {
        return nil, fmt.Errorf("failed to write thread count: %w", err)
    }

    // StartTime (unix seconds, or 0)
    var startUnix uint64 = 0
    if !w.StartTime.IsZero() {
        startUnix = uint64(w.StartTime.Unix())
    }
    if err := binary.Write(&buf, binary.BigEndian, startUnix); err != nil {
        return nil, fmt.Errorf("failed to write start time: %w", err)
    }

    // CPUTime (seconds, or 0)
    var cpuSeconds uint64 = 0
    if w.CPUTime > 0 {
        cpuSeconds = uint64(w.CPUTime.Seconds())
    }
    if err := binary.Write(&buf, binary.BigEndian, cpuSeconds); err != nil {
        return nil, fmt.Errorf("failed to write cpu time: %w", err)
    }

    // IsResponding
    if w.IsResponding {
        buf.WriteByte(0x01)
    } else {
        buf.WriteByte(0x00)
    }

    // MainWindowTitle
    if err := writeStr(&buf, w.MainWindowTitle); err != nil {
        return nil, fmt.Errorf("failed to write main window title: %w", err)
    }

    return buf.Bytes(), nil
}

// WSNProcessKill represents a request/response to terminate a process identified by PID.
type WSNProcessKill struct {
    PID uint64
}

func WSNProcessKillFromBytes(data []byte) (*WSNProcessKill, error) {
    if len(data) < 8 {
        return nil, fmt.Errorf("data too short for process kill")
    }
    pid := binary.BigEndian.Uint64(data[:8])
    return &WSNProcessKill{PID: pid}, nil
}

func (w *WSNProcessKill) ToData() ([]byte, error) {
    buf := make([]byte, 8)
    binary.BigEndian.PutUint64(buf, w.PID)
    return buf, nil
}

// WSNProcessList represents a request for the running process list. It carries no additional data.
type WSNProcessList struct{}

func WSNProcessListFromBytes(data []byte) (*WSNProcessList, error) {
    // No payload expected.
    if len(data) != 0 {
        // Not fatal, but warn.
    }
    return &WSNProcessList{}, nil
}

func (w *WSNProcessList) ToData() ([]byte, error) {
    return []byte{}, nil
}

// WSNProcessSTD represents data sent to the client containing stdout/stderr from a process.
// OutType distinguishes between STDOUT (0), STDERR (1), etc.
type WSNProcessSTD struct {
    OutType uint32
    Data    string
}

func WSNProcessSTDFromBytes(data []byte) (*WSNProcessSTD, error) {
    return WSNProcessSTDFromBuffer(bytes.NewReader(data))
}

func WSNProcessSTDFromBuffer(buf *bytes.Reader) (*WSNProcessSTD, error) {
    var outType uint32
    if err := binary.Read(buf, binary.BigEndian, &outType); err != nil {
        return nil, fmt.Errorf("failed to read outtype: %w", err)
    }
    s, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read data string: %w", err)
    }
    return &WSNProcessSTD{OutType: outType, Data: s}, nil
}

func (w *WSNProcessSTD) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := binary.Write(&buf, binary.BigEndian, w.OutType); err != nil {
        return nil, fmt.Errorf("failed to write outtype: %w", err)
    }
    if err := writeStr(&buf, w.Data); err != nil {
        return nil, fmt.Errorf("failed to write data string: %w", err)
    }
    return buf.Bytes(), nil
}

// WSNProcessStart represents a request to start a process with given command and arguments.
type WSNProcessStart struct {
    Command   string
    Arguments string
}

func WSNProcessStartFromBytes(data []byte) (*WSNProcessStart, error) {
    return WSNProcessStartFromBuffer(bytes.NewReader(data))
}

func WSNProcessStartFromBuffer(buf *bytes.Reader) (*WSNProcessStart, error) {
    cmd, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read command: %w", err)
    }
    args, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read arguments: %w", err)
    }
    return &WSNProcessStart{Command: cmd, Arguments: args}, nil
}

func (w *WSNProcessStart) ToData() ([]byte, error) {
    var buf bytes.Buffer
    if err := writeStr(&buf, w.Command); err != nil {
        return nil, fmt.Errorf("failed to write command: %w", err)
    }
    if err := writeStr(&buf, w.Arguments); err != nil {
        return nil, fmt.Errorf("failed to write arguments: %w", err)
    }
    return buf.Bytes(), nil
} 