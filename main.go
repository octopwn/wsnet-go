package main

import (
	"flag"
	"github.com/octopwn/wsnet-go/wsnet"
)

func main() {
	port := flag.Int("port", 8700, "Port to listen on")
	address := flag.String("address", "localhost", "Address to bind")
	uriPath := flag.String("uri-path", "", "URI path (or UUID) for WebSocket connection")
	disableSecurity := flag.Bool("disable-security", false, "Disable TLS security")
	flag.Parse()

	// Start the WebSocket server
	wsnet.StartWebsocketServer(disableSecurity, address, port, uriPath)
}
