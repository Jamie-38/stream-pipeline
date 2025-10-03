package main

import (
	"context"
	"strings"

	"github.com/gorilla/websocket"
)

func StartReader(ctx context.Context, conn *websocket.Conn, writerCh chan<- string, readCh chan<- string) error {
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		// One WebSocket frame can contain multiple IRC lines; split.
		for _, line := range strings.Split(string(payload), "\r\n") {
			if line == "" {
				continue
			}
			// keepalive
			if strings.HasPrefix(line, "PING") {
				writerCh <- "PONG :tmi.twitch.tv\r\n"
				continue
			}
			readCh <- line
		}
	}
}
