package wsnet

import (
	"bytes"
	"fmt"
	"strconv"
)

type WSNGetSequenceNoReply struct {
	Status  int
	EncData []byte
}

func WSNGetSequenceNoReplyFromBytes(b []byte) (*WSNGetSequenceNoReply, error) {
	buf := bytes.NewReader(b)
	var reply WSNGetSequenceNoReply

	// 1. Read Status
	status, err := readStr(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read Status: %w", err)
	}
	reply.Status, err = strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Status to int: %w", err)
	}

	// 2. Read EncData
	encData, err := readBytes(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read EncData: %w", err)
	}
	reply.EncData = encData

	return &reply, nil
}

func (w *WSNGetSequenceNoReply) ToData() ([]byte, error) {
	var buf bytes.Buffer

	// 1. Write Status
	statusStr := strconv.Itoa(w.Status)

	if err := writeStr(&buf, statusStr); err != nil {
		return nil, fmt.Errorf("failed to write Status: %w", err)
	}

	// 2. Write EncData
	if err := writeBytes(&buf, w.EncData); err != nil {
		return nil, fmt.Errorf("failed to write EncData: %w", err)
	}

	return buf.Bytes(), nil
}
