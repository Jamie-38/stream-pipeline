package main

import (
	"fmt"
	"os"

	"github.com/gorilla/websocket"
)

func TwitchWebsocket(token string, username string) (*websocket.Conn, error) {

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(os.Getenv("TWITCH_IRC_URI"), nil)
	if err != nil {
		fmt.Println(err)
	}

	msg := fmt.Sprintf("PASS oauth:%s\r\n", token)
	conn.WriteMessage(websocket.TextMessage, []byte(msg))
	nick := fmt.Sprintf("NICK %s\r\n", username)
	conn.WriteMessage(websocket.TextMessage, []byte(nick))
	// conn.WriteMessage(websocket.TextMessage, []byte("JOIN #hasanabi\r\n"))

	return conn, nil
}
