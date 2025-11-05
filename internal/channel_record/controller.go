// internal/channel_record/controller.go
package channelrecord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

// Controller is a single-writer "actor" that owns the desired channel set.
// It consumes IRCCommand from controlCh, persists to JSON, and exposes a read-only snapshot.
type Controller struct {
	path            string // path to channels.json
	account         string // validated account name
	schema          int    // schema version; start at 1
	controlCh       <-chan types.IRCCommand
	updatesCh       chan struct{} // edge-trigger notify (len 1)
	mu              sync.RWMutex  // guards snap
	snap            snapshot      // immutable view for readers
	writeDebounceMs int           // optional small debounce window
}

// snapshot is the immutable read view returned to readers.
type snapshot struct {
	Version   uint64
	Account   string
	UpdatedAt time.Time
	Channels  []string // normalized, sorted, no '#'? (we keep '#', Twitch-style)
}

// NewController loads or initializes channels.json, validates account, and returns a controller.
// It does not start the loop; call Run(ctx).
func NewController(path string, expectedAccount string, controlCh <-chan types.IRCCommand) (*Controller, error) {
	if path == "" {
		return nil, errors.New("channelrecord: empty path")
	}
	if expectedAccount == "" {
		return nil, errors.New("channelrecord: empty expectedAccount")
	}

	c := &Controller{
		path:            path,
		account:         expectedAccount,
		schema:          1,
		controlCh:       controlCh,
		updatesCh:       make(chan struct{}, 1),
		writeDebounceMs: 150, // small coalescing window for bursts
	}

	// Load existing file if present; otherwise create an empty set for this account.
	onDisk, err := loadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load channels file: %w", err)
	}

	var desired map[string]struct{}
	if err == nil {
		// File existed: validate account
		if onDisk.Account != "" && onDisk.Account != expectedAccount {
			return nil, fmt.Errorf("channels file account %q != expected %q", onDisk.Account, expectedAccount)
		}
		desired = sliceToSet(onDisk.Channels)
		c.schema = onDisk.Schema
	} else {
		desired = make(map[string]struct{})
	}

	// Build initial immutable snapshot
	chans := setToSortedSlice(desired)
	c.snap = snapshot{
		Version:   1,
		Account:   expectedAccount,
		UpdatedAt: time.Now().UTC(),
		Channels:  chans,
	}

	// If file didn't exist, persist initial empty set for the account.
	if errors.Is(err, os.ErrNotExist) {
		if err := c.writeFile(c.snap); err != nil {
			return nil, fmt.Errorf("initialize channels file: %w", err)
		}
	}

	return c, nil
}

// Run is the actor loop. It consumes HTTP intents from controlCh, updates the desired set,
// persists atomically (debounced), and notifies readers via Updates().
func (c *Controller) Run(ctx context.Context) error {
	// Local mutable working set (only visible inside Run).
	desired := sliceToSet(c.readSnap().Channels)
	version := c.readSnap().Version

	dirty := false
	var debounce <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case cmd := <-c.controlCh:
			// Normalize channel key
			ch, ok := normalizeChannel(cmd.Channel)
			if !ok {
				// bad input; ignore silently or log upstream
				continue
			}

			switch cmd.Op {
			case "JOIN":
				if _, exists := desired[ch]; !exists {
					desired[ch] = struct{}{}
					dirty = true
					if debounce == nil {
						debounce = time.After(time.Duration(c.writeDebounceMs) * time.Millisecond)
					}
				}
			case "PART":
				if _, exists := desired[ch]; exists {
					delete(desired, ch)
					dirty = true
					if debounce == nil {
						debounce = time.After(time.Duration(c.writeDebounceMs) * time.Millisecond)
					}
				}
			default:
				// unknown op; ignore or log
			}

		case <-debounce:
			if dirty {
				version++
				newSnap := snapshot{
					Version:   version,
					Account:   c.account,
					UpdatedAt: time.Now().UTC(),
					Channels:  setToSortedSlice(desired),
				}

				// Persist first; if it fails, surface error (let supervisor restart).
				if err := c.writeFile(newSnap); err != nil {
					return err
				}
				// Publish new snapshot for readers.
				c.writeSnap(newSnap)
				c.nonBlockingNotify()
			}
			dirty = false
			debounce = nil
		}
	}
}

// Snapshot returns the latest immutable view.
// It copies the channel slice so callers cannot mutate internal state.
func (c *Controller) Snapshot() (version uint64, channels []string, updatedAt time.Time, account string) {
	s := c.readSnap()
	cp := make([]string, len(s.Channels))
	copy(cp, s.Channels)
	return s.Version, cp, s.UpdatedAt, s.Account
}

// Updates is a best-effort "changed" ping channel.
// Readers can select on it to wake up promptly, then call Snapshot().
func (c *Controller) Updates() <-chan struct{} {
	return c.updatesCh
}

// --------------- helpers ---------------

func (c *Controller) readSnap() snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snap
}

func (c *Controller) writeSnap(s snapshot) {
	c.mu.Lock()
	c.snap = s
	c.mu.Unlock()
}

func (c *Controller) nonBlockingNotify() {
	select {
	case c.updatesCh <- struct{}{}:
	default:
		// drop if full; it's just a ping
	}
}

func normalizeChannel(raw string) (string, bool) {
	// Trim, lowercase, ensure leading '#'
	if raw == "" {
		return "", false
	}
	// minimal normalization — you can extend with stricter validation if desired
	// strip spaces
	var s string
	for i := 0; i < len(raw); i++ {
		if raw[i] != ' ' && raw[i] != '\t' && raw[i] != '\n' && raw[i] != '\r' {
			s = raw[i:]
			break
		}
	}
	if s == "" {
		return "", false
	}
	// lowercase
	b := make([]byte, 0, len(s)+1)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 'A' && ch <= 'Z' {
			ch = ch - 'A' + 'a'
		}
		b = append(b, ch)
	}
	if b[0] != '#' {
		b = append([]byte{'#'}, b...)
	}
	return string(b), true
}

func sliceToSet(xs []string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		if ch, ok := normalizeChannel(x); ok {
			m[ch] = struct{}{}
		}
	}
	return m
}

func setToSortedSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (c *Controller) writeFile(s snapshot) error {
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp := c.path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	onDisk := types.Channels{
		Schema:    c.schema,
		Account:   c.account,
		UpdatedAt: s.UpdatedAt,
		Channels:  s.Channels,
	}
	if err := enc.Encode(&onDisk); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode json: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("fsync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("rename tmp→final: %w", err)
	}
	// (Optional) fsync the directory for extra durability — not shown here.
	return nil
}

func loadFile(path string) (types.Channels, error) {
	var v types.Channels
	b, err := os.ReadFile(path)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return v, fmt.Errorf("decode %s: %w", path, err)
	}
	return v, nil
}
