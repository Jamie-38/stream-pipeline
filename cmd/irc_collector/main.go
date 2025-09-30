package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Jamie-38/stream-pipeline/internal/config"
	"github.com/Jamie-38/stream-pipeline/internal/oauth"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

// func Write_kafka(w *kafka.Writer) {
// 	err := w.WriteMessages(context.Background(),
// 		kafka.Message{
// 			Key:   []byte("Key-A"),
// 			Value: []byte("Hello World!"),
// 		},
// 	)
// 	if err != nil {
// 		log.Fatal("failed to write messages:", err)
// 	}

// 	if err := w.Close(); err != nil {
// 		log.Fatal("failed to close writer:", err)
// 	}
// }

func New_Writer() *kafka.Writer {
	w := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9094"),
		Topic:    "test-topic",
		Balancer: &kafka.LeastBytes{},
	}
	return w
}

type IRCCommand struct {
	Op      string // "JOIN", "PART", etc.
	Channel string // e.g., "#chess"
}

type APIController struct {
	controlCh chan IRCCommand
}

func (api *APIController) Join(w http.ResponseWriter, r *http.Request) {
	ch := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("channel")))
	if ch == "" {
		http.Error(w, "Missing channel parameter", http.StatusBadRequest)
		return
	}

	api.controlCh <- IRCCommand{Op: "JOIN", Channel: "#" + ch}
	w.Write([]byte("Queued join for channel: " + ch))
}

func New_http_controller(controlCh chan IRCCommand) {
	api := &APIController{controlCh: controlCh}
	mux := http.NewServeMux()
	mux.HandleFunc("/join", api.Join)

	host := os.Getenv("HTTP_API_HOST")
	port := os.Getenv("HTTP_API_PORT")
	address := net.JoinHostPort(host, port)

	log.Printf("http_api server listening on :%s", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatal(err)
	}
}

func Control_scheduler(ctx context.Context, controlCh <-chan IRCCommand, writerCh chan<- string) {
	for {
		select {
		case cmd := <-controlCh:
			switch cmd.Op {
			case "JOIN":
				writerCh <- fmt.Sprintf("JOIN %s\r\n", cmd.Channel)
			case "PART":
				writerCh <- fmt.Sprintf("PART %s\r\n", cmd.Channel)
			default:
				log.Printf("unknown IRC command: %+v", cmd)
			}
		case <-ctx.Done():
			return
		}
	}
}

func IRCWriter(ctx context.Context, conn *websocket.Conn, writerCh <-chan string) {
	for {
		select {
		case line := <-writerCh:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				log.Println("irc write error:", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func KafkaProducer(ctx context.Context, writer *kafka.Writer, readerCh <-chan string) {
	for {
		select {
		case line := <-readerCh:
			err := writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte("raw"),
				Value: []byte(line),
			})
			if err != nil {
				log.Println("kafka write error:", err)
				// optionally continue / backoff depending on your policy
			}

		case <-ctx.Done():
			return
		}
	}
}

// func print_reader(readerCh <-chan string) {
// 	for x := range readerCh {
// 		fmt.Println(x)
// 	}
// }

func main() {
	config.LoadEnv()

	account := config.LoadAccount()
	token := oauth.LoadTokenJSON()

	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create control and signal channel
	controlCh := make(chan IRCCommand, 100)
	sigCh := make(chan os.Signal, 1)

	// create irc writer and reader channel
	writerCh := make(chan string, 100)
	readerCh := make(chan string, 1000)

	// start http api_controller - go
	go New_http_controller(controlCh)

	// process control channel
	go Control_scheduler(ctx, controlCh, writerCh)

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
	w := New_Writer()
	defer w.Close()

	go KafkaProducer(ctx, w, readerCh)

	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	log.Println("shutting down...")
	cancel()

	// StartReader(conn, func(msg string) {
	// 	fmt.Printf("<<< %s\n", msg)
	// })

	// conn, err := TwitchWebsocket(token.AccessToken, account.Name)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer conn.Close()

	// StartReader(conn, func(msg string) {
	// 	fmt.Printf("<<< %s\n", msg)
	// })

	// w := New_Writer()
	// defer w.Close()

	// for {
	// 	_, p, read_err := conn.ReadMessage()
	// 	if read_err != nil {
	// 		log.Println(read_err)
	// 	}

	// 	write_err := w.WriteMessages(context.Background(),
	// 		kafka.Message{
	// 			Key:   []byte("Key-A"),
	// 			Value: []byte(p),
	// 		},
	// 	)
	// 	if write_err != nil {
	// 		log.Fatal("failed to write messages:", write_err)
	// 	}
	// }
}
