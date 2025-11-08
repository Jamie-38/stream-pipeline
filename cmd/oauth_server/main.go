// cmd/oauth_server/main.go
package main

import (
	"context"
	// "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Jamie-38/stream-pipeline/internal/config"
	"github.com/Jamie-38/stream-pipeline/internal/healthcheck"
	"github.com/Jamie-38/stream-pipeline/internal/oauth"
	"github.com/Jamie-38/stream-pipeline/internal/observe"
)

func main() {
	lg := observe.C("oauth_server")

	if err := config.LoadEnv(); err != nil {
		lg.Warn("env file not loaded", "err", err)
	}

	mux := http.NewServeMux()
	probe := healthcheck.New("oauth_server")
	probe.Register(mux)
	probe.SetNotReady()

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
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		// log.Fatalf("oauth_server: listen error: %v", err)
		lg.Error("listen error", "err", err, "port", port)
		os.Exit(1)
	}

	go func() {
		// log.Printf("oauth_server: listening on :%s", port)
		lg.Info("listening", "addr", ":"+port)
		probe.SetReady() // now we know the port is bound
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			// log.Printf("oauth_server: serve error: %v", err)
			lg.Error("serve error", "err", err)
			os.Exit(1)
		}
	}()

	// Trap SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	// log.Println("oauth_server: shutdown signal received")
	lg.Info("shutdown signal received", "port", port)

	// Give in-flight requests time to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop accepting new connections, drain keep-alives, finish in-flight.
	if err := srv.Shutdown(ctx); err != nil {
		// log.Printf("oauth_server: graceful shutdown did not complete: %v", err)
		lg.Error("graceful shutdown did not complete", "err", err)
		// Best-effort hard close if needed:
		if cerr := srv.Close(); cerr != nil {
			// log.Printf("oauth_server: forced close error: %v", cerr)
			lg.Error("forced close error", "err", cerr)
		}
	} else {
		lg.Info("shut down cleanly", "port", port)
	}
}
