package main

import (
	"flag"

	"github.com/octopwn/wsnet-go/wsnet"
)

func main() {
	port := flag.Int("port", 8700, "Port to listen on")
	address := flag.String("address", "localhost", "Address to bind to")
	uriPath := flag.String("uri-path", "<RANDOM>", "URI path (or UUID) for WebSocket connection")
	ssl := flag.Bool("ssl", false, "Enable SSL/TLS")
	flag.Parse()

	// inforeply is static but requires domain lookups, so we build it here once
	// and pass it to the WebSocket handler
	infoReply, err := wsnet.BuildGetInfoReply()
	if err != nil {
		panic(err)
	}
	// Start the WebSocket server
	wsnet.StartWebsocketServer(ssl, address, port, uriPath, infoReply)
}
