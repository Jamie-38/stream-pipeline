package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func LoadTokenJSON() types.Token {
	jsonFile, err := os.Open("tokens/default.token.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Opened Token JSON")
	fmt.Printf("%s", jsonFile.Name())
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var token types.Token
	json.Unmarshal(byteValue, &token)

	return token
}
