package httpapi

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func (api *APIController) Join(w http.ResponseWriter, r *http.Request) {
	ch := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("channel")))
	if ch == "" {
		http.Error(w, "Missing channel parameter", http.StatusBadRequest)
		return
	}

	api.ControlCh <- types.IRCCommand{Op: "JOIN", Channel: "#" + ch}
	w.Write([]byte("Queued join for channel: " + ch))
}

// func NewHTTPController(controlCh chan types.IRCCommand) {
// 	api := &APIController{ControlCh: controlCh}
// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/join", api.Join)

// 	host := os.Getenv("HTTP_API_HOST")
// 	port := os.Getenv("HTTP_API_PORT")
// 	address := net.JoinHostPort(host, port)

// 	log.Printf("http_api server listening on :%s", address)
// 	if err := http.ListenAndServe(address, mux); err != nil {
// 		log.Fatal(err)
// 	}
// }

func Run(ctx context.Context, controlCh chan types.IRCCommand) error {
	api := &APIController{ControlCh: controlCh}

	mux := http.NewServeMux()
	mux.HandleFunc("/join", api.Join)

	host := os.Getenv("HTTP_API_HOST")
	port := os.Getenv("HTTP_API_PORT")
	address := net.JoinHostPort(host, port)

	srv := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("http_api: listening on %s", address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return err
		}
		return nil
	case err := <-errCh:
		return err // surfacing serve errors to the caller
	}
}
