// internal/healthcheck/health.go
package healthcheck

import (
	"net/http"
	"sync/atomic"
	"time"

	"log/slog"

	"github.com/Jamie-38/stream-pipeline/internal/observe"
)

type Probe struct {
	ready int32 // 0 = not ready, 1 = ready
	lg    *slog.Logger
}

// New creates a health probe with a component-scoped logger.
func New(component string) *Probe {
	return &Probe{
		lg: observe.C("healthcheck").With("component", component),
	}
}

// Register wires /healthz (always 200) and /readyz (200 when ready, 503 otherwise).
func (p *Probe) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		// no per-request logs
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if atomic.LoadInt32(&p.ready) == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
			return
		}
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	})
}

// SetReady flips to ready and logs the transition once.
func (p *Probe) SetReady() {
	prev := atomic.SwapInt32(&p.ready, 1)
	if prev != 1 {
		p.lg.Info("readiness transition", "from", "not_ready", "to", "ready", "ts", time.Now().UTC())
	}
}

// SetNotReady flips to not-ready and logs the transition once.
func (p *Probe) SetNotReady() {
	prev := atomic.SwapInt32(&p.ready, 0)
	if prev != 0 {
		p.lg.Info("readiness transition", "from", "ready", "to", "not_ready", "ts", time.Now().UTC())
	}
}
