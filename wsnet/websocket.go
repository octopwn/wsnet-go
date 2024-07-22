package wsnet

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}



func WsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	ch := ClientHandler{}

	// Create a function that sends messages to the client
	ch.OnConnect(func(msg WSPacket) {
		data, err := serializeMessage(msg)
		if err != nil {
			log.Println(err)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Println(err)
			return
		}
	})


	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		if messageType != websocket.BinaryMessage {
			log.Println("Received non-binary message")
			continue
		}

		message, err := parseMessage(msg)
		if err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		fmt.Printf("Received: %+v\n", message)
		ch.OnMessage(message)
	}
}