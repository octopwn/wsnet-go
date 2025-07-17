// client_tailscale.go
//go:build tailscale && tailscaleclient
// +build tailscale,tailscaleclient

package main

import (
    "flag"
    "log"

    "github.com/octopwn/wsnet-go/wsnet"
)

func main() {
    url := flag.String("url", "", "WebSocket server URL to connect to (e.g., ws://server.tailnet:80/uuid)")
    hostname := flag.String("hostname", "wsnet-client", "hostname to use in the tailnet")
    authKey := flag.String("auth-key", "", "Tailscale auth key (optional). If omitted, TS_AUTH_KEY env var is used")
    flag.Parse()

    if *url == "" {
        log.Fatal("you must specify a -url to connect to")
    }

    wsnet.StartTailscaleWebsocketClient(url, hostname, authKey)
} 