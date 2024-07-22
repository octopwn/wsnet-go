

package wsnet
import (
	"bufio"
	"net"
	"fmt"
	"strconv"
)

type ClientHandler struct {
	send func(WSPacket)
	connections map[string]net.Conn

}

// Initialize the ClientHandler struct
func NewClientHandler() *ClientHandler {
	return &ClientHandler{
		connections: make(map[string]net.Conn),
	}
}

func (c *ClientHandler) OnConnect(sendfnt func(WSPacket)) {
	fmt.Println("Client connected")
	c.send = sendfnt
}

func (c *ClientHandler) OnDisconnect() {
	fmt.Println("Client disconnected")
	for token, conn := range c.connections {
		conn.Close()
		delete(c.connections, token)
	}
}

func (c *ClientHandler) OnMessage(message WSPacket) {
	fmt.Printf("Received: %+v\n", message)
	switch message.CmdType {
	case 7:
		{
			fmt.Println("Socket Data message received")
			if conn, ok := c.connections[fmt.Sprintf("%x", message.Token)]; ok {
				conn.Write(message.Data.([]byte))
			}
		}

	case 5:
		{
			fmt.Println("Socket Connect message received")
			if connData, ok := message.Data.(*WSNConnect); ok {
				if(connData.Bind) {
					fmt.Println("Bind not supported!")
					return
				} 
			
				fmt.Println("TCP Protocol")
				// Connect to the TCP server
				// Create a new TCP connection
				fmt.Println("Connecting to TCP server at IP:", connData.IP, "Port:", connData.Port)
				conn, err := net.Dial("tcp", connData.IP + ":" + strconv.Itoa(int(connData.Port)))
				if err != nil {
					fmt.Println("Error connecting to TCP server:", err)
					c.sendError(message.Token, err)
					return
				}

				tokenStr := fmt.Sprintf("%x", message.Token)
				c.connections[tokenStr] = conn
				fmt.Println("Connected to TCP server")

				// Start a goroutine to handle incoming data
				go c.handleIncomingData(message.Token, tokenStr, conn, c)
				c.sendContinue(message.Token)

			}
			
				fmt.Println("UDP Protocol")
			
		
		
		
		}
	}
}

func (c *ClientHandler) sendOK(token [16]byte) {
	okPacket := CreateOKPacket(token)
	c.send(okPacket)
}

func (c *ClientHandler) sendError(token [16]byte, err error) {
	fmt.Println("Error:", err)
	// Create an error packet
	byteErr := []byte(err.Error())
	errorPacket := WSPacket{
		Length: (uint32)(22 + len(byteErr)),
		CmdType: 1,
		Token: token,
		Data: byteErr,
	}
	c.send(errorPacket)
}

func (c *ClientHandler) sendSocketData(token [16]byte, data []byte) {
	dataPacket := WSPacket{
		Length: (uint32)(22 + len(data)),
		CmdType: 7,
		Token: token,
		Data: data,
	}
	c.send(dataPacket)
}

func (c *ClientHandler) sendContinue(token [16]byte) {
	continuePacket := CreateContinuePacket(token)
	c.send(continuePacket)
}

// Method to handle incoming data
func (c *ClientHandler) handleIncomingData(token [16]byte, tokenStr string, conn net.Conn, ch *ClientHandler) {
	reader := bufio.NewReader(conn)
	for {
		// Create a buffer to store the incoming data
		buffer := make([]byte, 65535) // Adjust the buffer size as needed
		n, err := reader.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			// Handle error (e.g., remove connection from map, close connection, etc.)
			conn.Close()
			delete(c.connections, tokenStr)
			ch.sendOK(token)
			break
		}
		// Process the received bytes
		receivedData := buffer[:n]
		fmt.Printf("Received data on connection %s: %x\n", tokenStr, receivedData)
		// You can add further processing of the receivedData as needed
		// Send the received data to the client
		ch.sendSocketData(token, receivedData)
	}
}
