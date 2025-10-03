package httpapi

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"

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

func New_http_controller(controlCh chan types.IRCCommand) {
	api := &APIController{ControlCh: controlCh}
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
