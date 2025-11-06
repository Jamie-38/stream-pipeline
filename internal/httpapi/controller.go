package httpapi

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Jamie-38/stream-pipeline/internal/healthcheck"
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

func Run(ctx context.Context, controlCh chan types.IRCCommand) error {
	api := &APIController{ControlCh: controlCh}

	mux := http.NewServeMux()
	healthcheck.Register(mux)

	mux.HandleFunc("/join", api.Join)

	healthcheck.SetReady()

	hostEnv := os.Getenv("HTTP_API_HOST")
	host := strings.TrimSpace(hostEnv)
	if host == "" {
		return fmt.Errorf("HTTP_API_HOST missing")
	}

	portEnv := os.Getenv("HTTP_API_PORT")
	portString := strings.TrimSpace(portEnv)
	if portString == "" {
		return fmt.Errorf("HTTP_API_PORT missing")
	}
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		return fmt.Errorf("HTTP_API_PORT parse error")
	}
	if port <= 1 || port >= 65535 {
		return fmt.Errorf("HTTP_API_PORT out of bounds")
	}

	address := net.JoinHostPort(host, portEnv)

	srv := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("http_api: listen error on %s: %w", address, err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("http_api: listening on %s", address)
		healthcheck.SetReady() // only after bind succeeds
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
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
