// websocket_tailscale.go
//go:build tailscale
// +build tailscale

package wsnet

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"tailscale.com/tsnet"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16384,
	WriteBufferSize: 16384,
}

func StartTailscaleWebsocketServer(address *string, hostname *string, uriPath *string, authKey *string, infoReply *WSNGetInfoReply) {
	var path string
	if *uriPath != "<RANDOM>" {
		path = *uriPath
	} else {
		// Generate random UUID if no URI path specified
		uuid4 := uuid.New().String()
		path = "/" + uuid4
	}

	fmt.Printf("Starting Tailscale WebSocket server on %s%s\n", *address, path)

	// Set up HTTP handler
	http.HandleFunc(path, WsHandler(infoReply))

	go func() {
		// Determine auth key: CLI flag overrides environment variable
		tsAuthKey := ""
		if authKey != nil && *authKey != "" {
			tsAuthKey = *authKey
		} else {
			tsAuthKey = os.Getenv("TS_AUTH_KEY")
		}

		if tsAuthKey == "" {
			fmt.Println("Tailscale auth key not provided. Provide it with -auth-key flag or TS_AUTH_KEY env variable. This is only required on the first run.")
		}

		s := &tsnet.Server{
			Hostname:  *hostname,
			AuthKey:   tsAuthKey,
			Ephemeral: true,
		}
		defer s.Close()

		ln, err := s.Listen("tcp", *address)
		if err != nil {
			log.Fatal(err)
		}
		defer ln.Close()

		// Start serving HTTP on the tsnet listener
		if err := http.Serve(ln, nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP Serve error: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")

}

func WsHandler(infoReply *WSNGetInfoReply) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		ch := NewClientHandler()

		// Example usage: use infoReply to send to client or log
		log.Printf("infoReply: %+v\n", infoReply)
		ch.inforeply = infoReply

		ch.OnConnect(func(msg WSPacket) {
			data, err := serializeMessage(msg)
			if err != nil {
				log.Println(err)
				ch.OnDisconnect()
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				log.Println(err)
				ch.OnDisconnect()
				return
			}
		})

		for {
			messageType, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				ch.OnDisconnect()
				return
			}

			if messageType != websocket.BinaryMessage {
				log.Println("Received non-binary message")
				continue
			}

			message, err := parseMessage(msg)
			if err != nil {
				log.Println("Error parsing message:", err)
				ch.OnDisconnect()
				continue
			}

			//fmt.Printf("Received: %+v\n", message)
			go ch.OnMessage(message)
		}
	}
}
