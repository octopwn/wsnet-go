package wsnet

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "strconv"
    "io"
)

func readBytes(r io.Reader) ([]byte, error) {
    var length uint32
    if err := binary.Read(r, binary.BigEndian, &length); err != nil {
        return nil, fmt.Errorf("failed to read bytes length: %w", err)
    }
    data := make([]byte, length)
    if _, err := io.ReadFull(r, data); err != nil {
        return nil, fmt.Errorf("failed to read bytes data: %w", err)
    }
    return data, nil
}

// writeBytes writes a byte slice prefixed with its 4-byte big-endian length.
func writeBytes(w io.Writer, data []byte) error {
    length := uint32(len(data))
    if err := binary.Write(w, binary.BigEndian, length); err != nil {
        return fmt.Errorf("failed to write bytes length: %w", err)
    }
    if _, err := w.Write(data); err != nil {
        return fmt.Errorf("failed to write bytes data: %w", err)
    }
    return nil
}

type WSNNTLMChallenge struct {
    AuthData   []byte
    CtxAttr    int
    TargetName string
}

func WSNNTLMChallengeFromBytes(b []byte) (*WSNNTLMChallenge, error) {
    buf := bytes.NewReader(b)
    var challenge WSNNTLMChallenge

    // 1. Read AuthData
    authData, err := readBytes(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read AuthData: %w", err)
    }
    challenge.AuthData = authData

    // 2. Read CtxAttr as string and convert to int
    ctxAttrStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read CtxAttr: %w", err)
    }
    ctxAttr, err := strconv.Atoi(ctxAttrStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert CtxAttr to int: %w", err)
    }
    challenge.CtxAttr = ctxAttr

    // 3. Read TargetName
    targetName, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read TargetName: %w", err)
    }
    challenge.TargetName = targetName

    return &challenge, nil
}

func (w *WSNNTLMChallenge) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // 1. Write AuthData
    if err := writeBytes(&buf, w.AuthData); err != nil {
        return nil, fmt.Errorf("failed to write AuthData: %w", err)
    }

    // 2. Write CtxAttr as string
    ctxAttrStr := strconv.Itoa(w.CtxAttr)
    if err := writeStr(&buf, ctxAttrStr); err != nil {
        return nil, fmt.Errorf("failed to write CtxAttr: %w", err)
    }

    // 3. Write TargetName
    if err := writeStr(&buf, w.TargetName); err != nil {
        return nil, fmt.Errorf("failed to write TargetName: %w", err)
    }

    return buf.Bytes(), nil
}


type WSNNTLMChallengeReply struct {
    Status    int
    CtxAttr   int
    AuthData  []byte
}



func WSNNTLMChallengeReplyFromBytes(b []byte) (*WSNNTLMChallengeReply, error) {
    buf := bytes.NewReader(b)
    var reply WSNNTLMChallengeReply

    // 1. Read Status
    status, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read Status: %w", err)
    }
    reply.Status, err = strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Status to int: %w", err)
	}

    // 2. Read CtxAttr as string and convert to int
    ctxAttrStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read CtxAttr: %w", err)
    }
    ctxAttr, err := strconv.Atoi(ctxAttrStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert CtxAttr to int: %w", err)
    }
    reply.CtxAttr = ctxAttr

    // 3. Read AuthData
    authData, err := readBytes(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read AuthData: %w", err)
    }
    reply.AuthData = authData

    return &reply, nil
}

func (w *WSNNTLMChallengeReply) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // 1. Write Status
	statusStr := strconv.Itoa(w.Status)
    if err := writeStr(&buf, statusStr); err != nil {
        return nil, fmt.Errorf("failed to write Status: %w", err)
    }

    // 2. Write CtxAttr as string
    ctxAttrStr := strconv.Itoa(w.CtxAttr)
    if err := writeStr(&buf, ctxAttrStr); err != nil {
        return nil, fmt.Errorf("failed to write CtxAttr: %w", err)
    }

    // 3. Write AuthData
    if err := writeBytes(&buf, w.AuthData); err != nil {
        return nil, fmt.Errorf("failed to write AuthData: %w", err)
    }

    return buf.Bytes(), nil
}

