

package wsnet
import (
	"bufio"
	"net"
	"fmt"
	"strconv"
	"time"
	"sync"
)

type ClientHandler struct {
	send func(WSPacket)
	connections map[string]net.Conn
	connectionsMu sync.Mutex       // Mutex for synchronizing access to connections
	outgoing    chan WSPacket       // Channel for outgoing messages

}

// Initialize the ClientHandler struct
func NewClientHandler() *ClientHandler {
	return &ClientHandler{
		connections: make(map[string]net.Conn),
		connectionsMu: sync.Mutex{}, 
		outgoing:    make(chan WSPacket, 1024), // Initialize the outgoing channel
	}
}

// Writer goroutine to handle outgoing messages
// This method is responsible to keep writing in a thread-safe way
// only this function can touch the outgoing channel on c.send
// do not call this function, only start it as a goroutine
func (c *ClientHandler) writePump() {
	for message := range c.outgoing {
		// Write the message to the WebSocket connection
		c.send(message)
	}
}

func (c *ClientHandler) OnConnect(sendfnt func(WSPacket)) {
	fmt.Println("Client connected")
	c.send = sendfnt
	go c.writePump() // Start the writer goroutine
}

func (c *ClientHandler) OnDisconnect() {
	fmt.Println("Client disconnected")
	close(c.outgoing) // Close the channel to stop the writer goroutine
	for token, conn := range c.connections {
		conn.Close()
		delete(c.connections, token)
	}
}

func (ch *ClientHandler) AddConnection(key string, conn net.Conn) {
	ch.connectionsMu.Lock()
	defer ch.connectionsMu.Unlock()
	ch.connections[key] = conn
}

func (ch *ClientHandler) RemoveConnection(key string) {
	ch.connectionsMu.Lock()
	defer ch.connectionsMu.Unlock()
	delete(ch.connections, key)
}


func (c *ClientHandler) OnMessage(message WSPacket) {
	fmt.Printf("Received: %+v\n", message)
	switch message.CmdType {
	case 0:
		{
			fmt.Println("OK message received")
			key := fmt.Sprintf("%x", message.Token)
			if _, ok := c.connections[key]; ok {
				c.RemoveConnection(key)
			}

			// if not in connections map, it could be in the future to be in the auth or file map but it's not implemented yet
		}
	case 1:
		{
			fmt.Println("Error message received")
			key := fmt.Sprintf("%x", message.Token)
			if conn, ok := c.connections[key]; ok {
				conn.Close()
				c.RemoveConnection(fmt.Sprintf("%x", message.Token))
			}
			//same as OK
		}
	case 7:
		{
			fmt.Println("Socket Data message received")
			key := fmt.Sprintf("%x", message.Token)
			if conn, ok := c.connections[key]; ok {
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
				timeout := time.Duration(1000 * time.Millisecond)
				conn, err := net.DialTimeout("tcp", connData.IP + ":" + strconv.Itoa(int(connData.Port)), timeout)
				if err != nil {
					fmt.Println("Error connecting to TCP server:", err)
					c.sendError(message.Token, err)
					return
				}

				tokenStr := fmt.Sprintf("%x", message.Token)
				c.AddConnection(tokenStr, conn)
				fmt.Println("Connected to TCP server")

				// Start a goroutine to handle incoming data
				go c.handleIncomingData(message.Token, tokenStr, conn, c)
				c.sendContinue(message.Token)

			} else {
				fmt.Println("UDP Protocol")
			}
		}
	}
}

func (c *ClientHandler) sendOK(token [16]byte) {
	okPacket := CreateOKPacket(token)
	c.outgoing <- okPacket
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

	c.outgoing <- errorPacket
}

func (c *ClientHandler) sendSocketData(token [16]byte, data []byte) {
	dataPacket := WSPacket{
		Length: (uint32)(22 + len(data)),
		CmdType: 7,
		Token: token,
		Data: data,
	}
	
	c.outgoing <- dataPacket
}

func (c *ClientHandler) sendContinue(token [16]byte) {
	continuePacket := CreateContinuePacket(token)

	c.outgoing <- continuePacket
}

// Method to handle incoming socket data
func (c *ClientHandler) handleIncomingData(token [16]byte, tokenStr string, conn net.Conn, ch *ClientHandler) {
	reader := bufio.NewReader(conn)
	for {
		// Create a buffer to store the incoming data
		buffer := make([]byte, 65535) // Adjust the buffer size as needed
		n, err := reader.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from connection:", err)
			// Handle error (e.g., remove connection from map, close connection, etc.)
			ch.RemoveConnection(tokenStr)
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
