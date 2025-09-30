package main

// import (
// 	"log"
// 	"net/http"
// 	"os"

// 	"github.com/Jamie-38/semantic-pipeline/internal/config"
// 	// "github.com/Jamie-38/semantic-pipeline/internal/oauth"
// )

// func main() {
// 	config.LoadEnv()

// 	mux := http.NewServeMux()
// 	// mux.HandleFunc("/", oauth.Index)
// 	// mux.HandleFunc("/callback", oauth.Callback)

// 	port := os.Getenv("http_api_SERVER_PORT")
// 	log.Printf("http_api server listening on :%s", port)
// 	if err := http.ListenAndServe(":"+port, mux); err != nil {
// 		log.Fatal(err)
// 	}
// }
