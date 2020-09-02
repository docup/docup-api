package main

import (
	"log"
	"os"
	"strconv"

	"github.com/docup/docup-api/gke/go-websocket-sample/external"
)

func main() {
	router := external.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	p, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	router.Run(int(p))
}
