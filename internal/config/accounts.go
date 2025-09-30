package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Jamie-38/stream-pipeline/internal/types"
)

func LoadAccount() types.Account {
	jsonFile, err := os.Open("accounts/percy.config.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Opened Account JSON")
	fmt.Printf("%s", jsonFile.Name())
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var account types.Account
	json.Unmarshal(byteValue, &account)

	return account
}
