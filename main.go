package main

import (
	"log"
	"net/http"
	"github.com/octopwn/wsnet-go/wsnet"
)

func main() {
	http.HandleFunc("/ws", wsnet.WsHandler)
	log.Println("WebSocket server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
