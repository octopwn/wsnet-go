
package wsnet

import (
	"encoding/binary"
	"fmt"
)

type WSPacket struct {
	Length  uint32
	CmdType uint16
	Token   [16]byte
	Data    []byte
}



func serializeMessage(message WSPacket) ([]byte, error) {
	buf := make([]byte, 22+len(message.Data))

	binary.BigEndian.PutUint32(buf[0:4], message.Length)
	binary.BigEndian.PutUint16(buf[4:6], message.CmdType)
	copy(buf[6:22], message.Token[:])
	copy(buf[22:], message.Data)

	return buf, nil
}

func parseMessage(data []byte) (WSPacket, error) {
	var message WSPacket

	if len(data) < 22 { // Minimum size: 4 (length) + 2 (cmdtype) + 16 (token)
		return message, fmt.Errorf("data too short")
	}

	message.Length = binary.BigEndian.Uint32(data[0:4])
	message.CmdType = binary.BigEndian.Uint16(data[4:6])
	copy(message.Token[:], data[6:22])

	if len(data) > 22 {
		message.Data = data[22:]
	} else {
		message.Data = []byte{}
	}

	return message, nil
}
