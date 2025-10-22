package main

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
)

func TwitchWebsocket(ctx context.Context, token, username, uri string) (*websocket.Conn, error) {
	d := websocket.Dialer{}
	conn, _, err := d.Dial(uri, nil)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	write := func(s string) error {
		return conn.WriteMessage(websocket.TextMessage, []byte(s))
	}

	// 1) Auth
	if err := write(fmt.Sprintf("PASS oauth:%s\r\n", token)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pass: %w", err)
	}
	if err := write(fmt.Sprintf("NICK %s\r\n", username)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("nick: %w", err)
	}

	// 2) Capabilities (batched)
	if err := write("CAP REQ :twitch.tv/tags twitch.tv/commands twitch.tv/membership\r\n"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("cap req: %w", err)
	}

	return conn, nil
}
