package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Jamie-38/stream-pipeline/internal/config"
	"github.com/Jamie-38/stream-pipeline/internal/httpapi"
	ircevents "github.com/Jamie-38/stream-pipeline/internal/irc_events"
	kstream "github.com/Jamie-38/stream-pipeline/internal/kafka"
	"github.com/Jamie-38/stream-pipeline/internal/oauth"
	"github.com/Jamie-38/stream-pipeline/internal/scheduler"
	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func main() {
	config.LoadEnv()

	account := config.LoadAccount()
	token := oauth.LoadTokenJSON()

	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create control and signal channel
	controlCh := make(chan types.IRCCommand, 100)
	sigCh := make(chan os.Signal, 1)

	// create irc writer and reader channel
	writerCh := make(chan string, 100)
	readerCh := make(chan string, 1000)

	parseCh := make(chan ircevents.Event, 1000)

	errCh := make(chan error, 1)

	// process control channel
	go scheduler.Control_scheduler(ctx, controlCh, writerCh)

	// dial
	conn, err := TwitchWebsocket(ctx, token.AccessToken, account.Name, os.Getenv("TWITCH_IRC_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	go func() { errCh <- httpapi.Run(ctx, controlCh) }()

	go func() { errCh <- StartReader(ctx, conn, writerCh, readerCh) }()

	go func() { errCh <- IRCWriter(ctx, conn, writerCh) }()

	// start  irc writer
	// go IRCWriter(ctx, conn, writerCh)

	// start irc reader
	// go StartReader(ctx, conn, writerCh, readerCh)

	// parse readerCh
	go Classify_line(ctx, readerCh, parseCh)

	// kafka producer
	w := kstream.NewWriter()
	defer w.Close()

	go kstream.KafkaProducer(ctx, w, parseCh)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Println("shutting down...")
	cancel()

}
