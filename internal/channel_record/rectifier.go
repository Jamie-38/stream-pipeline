package channelrecord

import (
	"context"
	"strings"
	"time"

	// ircevents "github.com/Jamie-38/stream-pipeline/internal/irc_events"
	"github.com/Jamie-38/stream-pipeline/internal/types"
)

type DesiredSnapshot interface {
	Snapshot() (version uint64, channels []string, updatedAt time.Time, account string)
	Updates() <-chan struct{}
}

type Config struct {
	TokensPerSecond float64
	Burst           int
	JoinTimeout     time.Duration
	BackoffMin      time.Duration
	BackoffMax      time.Duration
	Tick            time.Duration
}

func NewDefaultConfig() Config {
	return Config{
		TokensPerSecond: 0.5,
		Burst:           2,
		JoinTimeout:     30 * time.Second,
		BackoffMin:      2 * time.Second,
		BackoffMax:      60 * time.Second,
		Tick:            1 * time.Second,
	}
}

func Run(ctx context.Context, desired DesiredSnapshot, events <-chan types.MembershipEvent, out chan<- types.IRCCommand, cfg Config) error {
	r := &reconciler{
		desired:      desired,
		events:       events,
		out:          out,
		cfg:          cfg,
		state:        make(map[string]*chanState),
		tokenBucket:  newBucket(cfg.TokensPerSecond, cfg.Burst),
		lastDesiredV: 0,
	}
	return r.loop(ctx)
}

type phase int

const (
	Idle phase = iota
	Joining
	Joined
	Parting
	Error
)

type chanState struct {
	want      bool
	have      bool
	phase     phase
	lastTry   time.Time
	deadline  time.Time
	backoff   time.Duration
	nextTryAt time.Time
}

type reconciler struct {
	desired      DesiredSnapshot
	events       <-chan types.MembershipEvent
	out          chan<- types.IRCCommand
	cfg          Config
	state        map[string]*chanState
	tokenBucket  *bucket
	lastDesiredV uint64
}

func (r *reconciler) loop(ctx context.Context) error {
	tick := time.NewTicker(r.cfg.Tick)
	defer tick.Stop()

	updates := r.desired.Updates()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-tick.C:
			r.observeDesired()
			r.reconcile(time.Now())

		case <-updates:
			r.observeDesired()
			r.reconcile(time.Now())

		case evt := <-r.events:
			r.observeEvent(evt)
			r.reconcile(time.Now())
		}
	}
}

func (r *reconciler) observeDesired() {
	v, chans, _, _ := r.desired.Snapshot()
	if v == r.lastDesiredV {
		return
	}
	for name, s := range r.state {
		s.want = false
		_ = name
	}
	for _, ch := range chans {
		s := r.ensure(ch)
		s.want = true
	}
	r.lastDesiredV = v
}

func (r *reconciler) observeEvent(evt types.MembershipEvent) {
	switch evt.Op {
	case "JOIN":
		ch := strings.ToLower(evt.Channel)
		if !strings.HasPrefix(ch, "#") {
			ch = "#" + ch
		}
		s := r.ensure(ch)
		if !s.have {
			s.have = true
			s.phase = Joined
		}

	default:
		// ignore
	}
}

func (r *reconciler) reconcile(now time.Time) {

	for name, s := range r.state {
		if s.want {
			continue
		}
		if s.have && (s.phase == Idle || s.phase == Joined || (s.phase == Error && now.After(s.nextTryAt))) {
			if r.trySend(now, "PART", name, s) {
				continue
			}
		}

		r.maybeTimeout(now, s, "PART")
	}

	for name, s := range r.state {
		if !s.want {
			continue
		}
		if !s.have && (s.phase == Idle || s.phase == Error && now.After(s.nextTryAt)) {
			if r.trySend(now, "JOIN", name, s) {
				continue
			}
		}
		r.maybeTimeout(now, s, "JOIN")
	}
}

func (r *reconciler) trySend(now time.Time, op string, channel string, s *chanState) bool {
	if !r.tokenBucket.take(now) {
		return false
	}

	select {
	case r.out <- types.IRCCommand{Op: op, Channel: channel}:
		s.lastTry = now
		s.deadline = now.Add(r.cfg.JoinTimeout)
		if op == "JOIN" {
			s.phase = Joining
			if s.backoff == 0 {
				s.backoff = r.cfg.BackoffMin
			}
		} else {
			s.phase = Parting
			if s.backoff == 0 {
				s.backoff = r.cfg.BackoffMin
			}
		}
		return true
	default:
		r.tokenBucket.refund(now)
		return false
	}
}

func (r *reconciler) maybeTimeout(now time.Time, s *chanState, op string) {
	if (s.phase == Joining || s.phase == Parting) && now.After(s.deadline) {
		s.phase = Error
		s.nextTryAt = now.Add(s.backoff)
		s.backoff *= 2
		if s.backoff > r.cfg.BackoffMax {
			s.backoff = r.cfg.BackoffMax
		}
	}
}

func (r *reconciler) ensure(ch string) *chanState {
	if st, ok := r.state[ch]; ok {
		return st
	}
	st := &chanState{
		want:    false,
		have:    false,
		phase:   Idle,
		backoff: r.cfg.BackoffMin,
	}
	r.state[ch] = st
	return st
}

type bucket struct {
	rate       float64
	capacity   float64
	tokens     float64
	lastUpdate time.Time
}

func newBucket(tokensPerSec float64, burst int) *bucket {
	return &bucket{
		rate:       tokensPerSec,
		capacity:   float64(burst),
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

func (b *bucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastUpdate).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastUpdate = now
}

func (b *bucket) take(now time.Time) bool {
	b.refill(now)
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

func (b *bucket) refund(now time.Time) {
	b.refill(now)
	b.tokens += 1.0
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
}
