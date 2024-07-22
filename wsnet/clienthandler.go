

package wsnet
import "fmt"

type ClientHandler struct {
	send func(WSPacket)
}

func (c *ClientHandler) OnConnect(sendfnt func(WSPacket)) {
	fmt.Println("Client connected")
	c.send = sendfnt
}

func (c *ClientHandler) OnMessage(message WSPacket) {
	fmt.Printf("Received: %+v\n", message)
}

