// cmd/oauth_server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Jamie-38/stream-pipeline/internal/config"
	"github.com/Jamie-38/stream-pipeline/internal/oauth"
)

func main() {
	config.LoadEnv()

	mux := http.NewServeMux()
	mux.HandleFunc("/", oauth.Index)
	mux.HandleFunc("/callback", oauth.Callback)

	port := os.Getenv("OAUTH_SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		// ErrorLog:        log.New(...), // optional: custom logger
	}

	// Start the server in the background.
	go func() {
		log.Printf("oauth_server: listening on :%s", port)
		// http.ErrServerClosed is expected on Shutdown; treat other errors as fatal.
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("oauth_server: listen error: %v", err)
		}
	}()

	// Trap SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	log.Println("oauth_server: shutdown signal received")

	// Give in-flight requests time to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new connections, drain keep-alives, finish in-flight.
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("oauth_server: graceful shutdown did not complete: %v", err)
		// Best-effort hard close if needed:
		if cerr := srv.Close(); cerr != nil {
			log.Printf("oauth_server: forced close error: %v", cerr)
		}
	} else {
		log.Println("oauth_server: shut down cleanly")
	}
}
