package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func LoadChannels(path string) (types.Channels, error) {
	var chans types.Channels

	f, err := os.Open(path)
	if err != nil {
		return chans, fmt.Errorf("open account file %q: %w", path, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err := dec.Decode(&chans); err != nil {
		return chans, fmt.Errorf("decode account json %q: %w", path, err)
	}

	// Minimal validation to catch empty/misnamed fields early.
	if chans.Account == "" {
		return chans, fmt.Errorf("account %q missing required field: username/name", path)
	}
	return chans, nil
}
