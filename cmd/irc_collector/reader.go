package main

import (
	"context"
	"strings"

	"github.com/gorilla/websocket"
)

// func StartReader(conn *websocket.Conn, handler func(string)) error {
// 	for {
// 		_, p, err := conn.ReadMessage()
// 		if err != nil {
// 			log.Println(err)
// 			return err
// 		}
// 		handler(string(p))
// 	}
// }

func StartReader(ctx context.Context, conn *websocket.Conn, writerCh chan<- string, readCh chan<- string) error {
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			// When ctx is canceled you’ll likely see an error here because the conn is closed.
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
