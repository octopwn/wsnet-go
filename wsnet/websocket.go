package wsnet

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  16384,
	WriteBufferSize: 16384,
}


func StartWebsocketServer(disableSecurity *bool, address *string, port *int, uriPath *string, infoReply *WSNGetInfoReply) {
	// Determine URI path
	var path string
	if *disableSecurity {
		path = "/ws"
	} else {
		if *uriPath != "" {
			path = *uriPath
		} else {
			// Generate random UUID if no URI path specified
			uuid4 := uuid.New().String()
			path = "/" + uuid4
		}
	}

	// Set up HTTP handler
	http.HandleFunc(path, WsHandler(infoReply))

	// Start server
	addr := fmt.Sprintf("%s:%d", *address, *port)
	
	server := &http.Server{
		Addr: addr,
	}

	// Handle TLS (HTTPS) if security is enabled
	if !*disableSecurity {
		certFile := "server.crt"
		keyFile := "server.key"

		// Check if certificate and key files exist, otherwise generate new ones
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			err := generateCertificate(certFile, keyFile)
			if err != nil {
				log.Fatalf("Failed to generate certificate: %v", err)
			}
		}

		tlsConfig, err := loadTLSConfig(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to load TLS config: %v", err)
		}
		server.TLSConfig = tlsConfig

		log.Printf("WebSocket server (secure) started on %s", addr)
		go func() {
			fmt.Printf("Starting server on wss://%s%s\n", addr, path)
			err := server.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				log.Fatalf("ListenAndServeTLS: %v", err)
			}
		}()
	} else {
		log.Printf("WebSocket server (insecure) started on %s", addr)
		go func() {
			fmt.Printf("Starting server on ws://%s%s\n", addr, path)
			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				log.Fatalf("ListenAndServe: %v", err)
			}
		}()
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")

	if err := server.Shutdown(nil); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
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

            fmt.Printf("Received: %+v\n", message)
            go ch.OnMessage(message)
        }
    }
}
