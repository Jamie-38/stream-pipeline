package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	// channelrecord "github.com/Jamie-38/stream-pipeline/internal/channel_record"
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

	account, err := config.LoadAccount(os.Getenv("ACCOUNTS_PATH"))
	if err != nil {
		log.Fatalf("load account: %v", err)
	}

	channels, err := config.LoadChannels(os.Getenv("CHANNELS_PATH"))
	if err != nil {
		log.Fatalf("load channels: %v", err)
	}

	token, err := oauth.LoadTokenJSON(os.Getenv("TOKENS_PATH"))
	if err != nil {
		log.Fatalf("load token: %v", err)
	}

	// validate channels match account
	if account.Name != channels.Account {
		log.Fatalf("account name does not match channels account")
	}

	// ctx canceled by signal
	root, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// pipeline context derives from root
	g, ctx := errgroup.WithContext(root)

	// pipeline channels
	controlCh := make(chan types.IRCCommand, 100)
	// irceventCh := make(chan types.IRCCommand, 100)
	writerCh := make(chan string, 100)
	readerCh := make(chan string, 1000)
	parseCh := make(chan ircevents.Event, 1000)

	// connect (fail fast before goroutines)
	conn, err := TwitchWebsocket(ctx, token.AccessToken, account.Name, os.Getenv("TWITCH_IRC_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// kafka writer (lifecycle tied to main)
	w := kstream.NewWriter()
	defer w.Close()

	// ---- all stages run under errgroup ----

	// HTTP control plane
	g.Go(func() error { return httpapi.Run(ctx, controlCh) })

	// channel manager

	// IRC control scheduler (JOIN/PART -> writerCh)
	g.Go(func() error {
		scheduler.Control_scheduler(ctx, controlCh, writerCh)
		return nil
	})

	// IRC socket reader -> readerCh (and PING signals -> writerCh)
	g.Go(func() error { return StartReader(ctx, conn, writerCh, readerCh) })

	// Single writer to the socket
	g.Go(func() error { return IRCWriter(ctx, conn, writerCh) })

	// Parser: readerCh -> parseCh
	g.Go(func() error {
		Classify_line(ctx, readerCh, parseCh)
		return nil
	})

	// Kafka producer: parseCh -> Kafka
	g.Go(func() error {
		kstream.KafkaProducer(ctx, w, parseCh)
		return nil
	})

	// ---- wait for first error or signal ----
	if err := g.Wait(); err != nil {
		log.Printf("fatal pipeline error: %v", err)
	}
}
