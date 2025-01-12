package wsnet

import (
    "bytes"
    "fmt"
    "strconv"
)

type WSNSessionKeyReply struct {
    Status      int
    SessionKey  []byte
}

func WSNSessionKeyReplyFromBytes(b []byte) (*WSNSessionKeyReply, error) {
    buf := bytes.NewReader(b)
    var reply WSNSessionKeyReply

    // 1. Read Status
    status, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read Status: %w", err)
    }
    reply.Status, err = strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Status to int: %w", err)
	}


    // 2. Read SessionKey
    sessionKey, err := readBytes(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read SessionKey: %w", err)
    }
    reply.SessionKey = sessionKey

    return &reply, nil
}

func (w *WSNSessionKeyReply) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // 1. Write Status
	statusStr := strconv.Itoa(w.Status)

    if err := writeStr(&buf, statusStr); err != nil {
        return nil, fmt.Errorf("failed to write Status: %w", err)
    }

    // 2. Write SessionKey
    if err := writeBytes(&buf, w.SessionKey); err != nil {
        return nil, fmt.Errorf("failed to write SessionKey: %w", err)
    }

    return buf.Bytes(), nil
}

