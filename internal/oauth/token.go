package oauth

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func LoadTokenJSON(path string) (types.Token, error) {
	var tok types.Token

	f, err := os.Open(path)
	if err != nil {
		return tok, fmt.Errorf("open token file %q: %w", path, err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&tok); err != nil {
		return tok, fmt.Errorf("decode token json %q: %w", path, err)
	}

	// Minimal validation: presence and scope sanity (optional, tweak as needed).
	if tok.AccessToken == "" {
		return tok, fmt.Errorf("token %q missing access_token", path)
	}
	return tok, nil
}
