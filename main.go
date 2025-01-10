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

	// inforeply is static but requires domain lookups, so we build it here once
	// and pass it to the WebSocket handler
	infoReply, err := wsnet.BuildGetInfoReply()
	if err != nil {
		panic(err)
	}


	// Start the WebSocket server
	wsnet.StartWebsocketServer(disableSecurity, address, port, uriPath, infoReply)
}
