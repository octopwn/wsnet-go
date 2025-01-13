// main_tailscale.go
//go:build tailscale
// +build tailscale

package main

import (
	"flag"

	"github.com/octopwn/wsnet-go/wsnet"
)

func main() {
	address := flag.String("address", ":80", "Address and port to bind to")
	hostname := flag.String("hostname", "wsnet", "hostname to use in the tailnet")
	uriPath := flag.String("uri-path", "<RANDOM>", "URI path (or UUID) for WebSocket connection")
	flag.Parse()

	// inforeply is static but requires domain lookups, so we build it here once
	// and pass it to the WebSocket handler
	infoReply, err := wsnet.BuildGetInfoReply()
	if err != nil {
		panic(err)
	}
	// Start the WebSocket server
	wsnet.StartTailscaleWebsocketServer(address, hostname, uriPath, infoReply)
}
