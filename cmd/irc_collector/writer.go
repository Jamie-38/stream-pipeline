package main

import (
	"context"

	"github.com/gorilla/websocket"
)

func IRCWriter(ctx context.Context, conn *websocket.Conn, writerCh <-chan string) error {
	for {
		select {
		case line := <-writerCh:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
