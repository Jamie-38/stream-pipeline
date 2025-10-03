package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Jamie-38/stream-pipeline/internal/config"
	"github.com/Jamie-38/stream-pipeline/internal/httpapi"
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

	// start http api_controller - go
	go httpapi.New_http_controller(controlCh)

	// process control channel
	go scheduler.Control_scheduler(ctx, controlCh, writerCh)

	// dial
	conn, err := TwitchWebsocket(token.AccessToken, account.Name)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// start writer
	go IRCWriter(ctx, conn, writerCh)

	// start reader
	go StartReader(ctx, conn, writerCh, readerCh)

	// print irc stream - TEST
	// go print_reader(readerCh)

	// kafka producer
	w := kstream.New_Writer()
	defer w.Close()

	go kstream.KafkaProducer(ctx, w, readerCh)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Println("shutting down...")
	cancel()

}
