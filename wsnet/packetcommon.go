package wsnet

import (
	"bytes"
	"encoding/binary"
)

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
