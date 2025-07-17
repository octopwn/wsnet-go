// websocket_client_tailscale.go
//go:build tailscale
// +build tailscale

package wsnet

import (
    "context"
    "fmt"
    "log"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/gorilla/websocket"
    "tailscale.com/tsnet"
)

// StartTailscaleWebsocketClient starts a WebSocket client that connects over the
// Tailnet using tsnet. It mirrors the behaviour of StartTailscaleWebsocketServer
// but as an outbound client instead of an inbound server.
func StartTailscaleWebsocketClient(serverURL *string, hostname *string, authKey *string) {
    // Build the static info reply once; this is required when the peer sends a
    // GetInfo (cmd 8) packet.
    infoReply, err := BuildGetInfoReply()
    if err != nil {
        log.Fatalf("failed to build get-info reply: %v", err)
    }

    // Determine the Tailscale auth key: CLI flag overrides environment
    // variable, matching the server behaviour for consistency.
    tsAuthKey := ""
    if authKey != nil && *authKey != "" {
        tsAuthKey = *authKey
    } else {
        tsAuthKey = os.Getenv("TS_AUTH_KEY")
    }
    if tsAuthKey == "" {
        fmt.Println("Tailscale auth key not provided. Provide it with -auth-key flag or TS_AUTH_KEY env variable. This is only required on the first run.")
    }

    // Spin up a lightweight tsnet node so we can communicate on the tailnet.
    s := &tsnet.Server{
        Hostname:  *hostname,
        AuthKey:   tsAuthKey,
        Ephemeral: true,
    }
    defer s.Close()
    if _, err := s.Up(context.Background()); err != nil {
        log.Fatalf("tsnet up error: %v", err)
    }

    // Create a WebSocket dialer whose underlying network dials through tsnet so
    // that all traffic flows over the Tailnet rather than the physical
    // network.
    dialer := websocket.Dialer{
        NetDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
            return s.Dial(ctx, network, address)
        },
        Proxy:          http.ProxyFromEnvironment,
        ReadBufferSize:  16384,
        WriteBufferSize: 16384,
    }

    fmt.Printf("Connecting to WebSocket server %s\n", *serverURL)
    conn, _, err := dialer.Dial(*serverURL, nil)
    if err != nil {
        log.Fatalf("failed to dial websocket: %v", err)
    }
    defer conn.Close()

    // ClientHandler deals with multiplexed protocol packets.
    ch := NewClientHandler()
    ch.inforeply = infoReply

    // Register the send function so ClientHandler can write back to the server.
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

    // Reader loop – parse each incoming binary frame into a WSPacket and pass
    // it to the handler.
    go func() {
        for {
            messageType, msg, err := conn.ReadMessage()
            if err != nil {
                log.Println("websocket read:", err)
                ch.OnDisconnect()
                return
            }

            if messageType != websocket.BinaryMessage {
                log.Println("received non-binary message")
                continue
            }

            message, err := parseMessage(msg)
            if err != nil {
                log.Println("error parsing message:", err)
                ch.OnDisconnect()
                continue
            }

            fmt.Printf("Received: %+v\n", message)
            go ch.OnMessage(message)
        }
    }()

    // Graceful shutdown on SIGINT / SIGTERM.
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
    <-stop
    fmt.Println("Shutting down client…")
} 