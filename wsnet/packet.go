
package wsnet

import (
	"encoding/binary"
	"fmt"
)

type WSPacket struct {
	Length  uint32
	CmdType uint16
	Token   [16]byte
	Data    interface{}
}



func serializeReply(message WSPacket, cmdType uint16, data []byte) WSPacket {
	return WSPacket{Length: uint32(22 + len(data)), CmdType: cmdType, Token: message.Token, Data: data}
}

func serializeMessage(message WSPacket) ([]byte, error) {
	datalen := -1
	msgData := make([]byte, 0)
	if message.Data == nil {
		datalen = 0
	}
	if dataBytes, ok := message.Data.([]byte); ok {
		datalen = len(dataBytes)
		msgData = dataBytes
	}
	if datalen == -1 {
		// TODO: continue here. current implementation only supports byte slices, but there are other data types
		return nil, fmt.Errorf("Unknown data type")
	}


	buf := make([]byte, 22+datalen)
	binary.BigEndian.PutUint32(buf[0:4], message.Length)
	binary.BigEndian.PutUint16(buf[4:6], message.CmdType)
	copy(buf[6:22], message.Token[:])
	copy(buf[22:], msgData)

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

	if len(data) < 22 {
		return message, fmt.Errorf("Message data too short")
	}
	
	switch message.CmdType {
	case 0:
		{
			//OK
			return message, nil
		}
	case 1:
		{
			//error
			wsnErrParsed, err := WSNErrFromBytes(data[22:])
			if err != nil {
				fmt.Println("Error:", err)
				return message, err
			}
			message.Data = wsnErrParsed
			return message, nil
		}
	case 2:
		{
			// LOG
			// TODO
			return message, nil
		}
	case 3:
		{
			// STOP
			return message, nil
		}
	case 4:
		{
			// Continue
			return message, nil
		}
	case 5:
		{
			wsnConnectParsed, err := WSNConnectFromBytes(data[22:])
			if err != nil {
				fmt.Println("Error:", err)
				return message, err
			}
			message.Data = wsnConnectParsed
			return message, nil
		}
	case 6:
		{
			// Disconnect, not in use
			return message, nil
		}
	case 7:
		{
			// Data
			message.Data = data[22:]
			return message, nil
		}
	case 8:
		{
			// getinfo
			return message, nil
		}
	default:
		{
			return message, fmt.Errorf("Unknown command type: %d", message.CmdType)
		
		}
	} 		
}

//TODO: check this
func ReadString(data []byte, offset int) (string, int) {
	length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	return string(data[offset+4 : offset+4+length]), offset + 4 + length
}

func BoolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func CreateOKPacket(token [16]byte) WSPacket {
	return WSPacket{Length: 22, CmdType: 0, Token: token}
}

func CreateErrorPacket(token [16]byte, reason string, extra string) WSPacket {
	return WSPacket{Length: 0, CmdType: 1, Token: token, Data: NewWSNErr(reason, extra)}
}

func CreateContinuePacket(token [16]byte) WSPacket {
	return WSPacket{Length: 22, CmdType: 4, Token: token}
}

func CreateSDPacket(token [16]byte, data []byte) WSPacket {
	return WSPacket{Length: (uint32)(22 + len(data)) , CmdType: 7, Token: token, Data: data}
}

func CreateGetInfoPacket(token [16]byte, infoReply *WSNGetInfoReply) WSPacket {
	infodata, err := infoReply.ToData()
	if err != nil {
		fmt.Printf("Failed to serialize getinfo reply: %v", err)
		return WSPacket{Length: 22, CmdType: 8, Token: token}
	}
	return WSPacket{Length: 22, CmdType: 8, Token: token, Data: infodata}
}

