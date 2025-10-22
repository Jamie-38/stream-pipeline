package main

import (
	"context"
	"strings"

	"github.com/gorilla/websocket"
)

func StartReader(ctx context.Context, conn *websocket.Conn, writerCh chan<- string, readCh chan<- string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return err
			}
			for _, line := range strings.Split(string(payload), "\r\n") {
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "PING") {
					select {
					case writerCh <- "PONG :tmi.twitch.tv\r\n":
					case <-ctx.Done():
						return ctx.Err()
					}
					continue
				}
				select {
				case readCh <- line:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}
