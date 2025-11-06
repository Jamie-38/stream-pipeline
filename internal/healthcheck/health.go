package healthcheck

import (
	"net/http"
	"sync/atomic"
)

var ready int32 // 0 = not ready, 1 = ready

// Register wires /healthz (always OK) and /readyz (gates on SetReady).
func Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if atomic.LoadInt32(&ready) == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
			return
		}
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	})
}

func SetReady()    { atomic.StoreInt32(&ready, 1) }
func SetNotReady() { atomic.StoreInt32(&ready, 0) }
