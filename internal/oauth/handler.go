package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func Index(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	redirectURI := os.Getenv("TWITCH_REDIRECT_URI")

	authURL := fmt.Sprintf(
		"https://id.twitch.tv/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=chat:read",
		clientID, redirectURI)

	fmt.Fprintf(w, `<a href="%s">Click here to authenticate with Twitch</a>`, authURL)
}

func Callback(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	redirectURI := os.Getenv("TWITCH_REDIRECT_URI")
	tokenURL := "https://id.twitch.tv/oauth2/token"

	code := r.URL.Query().Get("code")
	if code == "" {
		fmt.Fprintf(w, "Error: No code received.")
		return
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		fmt.Fprintf(w, "Failed to post: %v", err)
		return
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var tokenData types.Token
	err = decoder.Decode(&tokenData)
	if err != nil {
		fmt.Fprintf(w, "Failed to parse response: %v", err)
		return
	}

	fmt.Fprintf(w, "%+v\n", tokenData)

	f, err := os.Create(fmt.Sprintf("tokens/%s.token.json", "default"))
	if err != nil {
		fmt.Fprintf(w, "Failed to write token file: %v", err)
		return
	}
	defer f.Close()

	json.NewEncoder(f).Encode(tokenData)
}
