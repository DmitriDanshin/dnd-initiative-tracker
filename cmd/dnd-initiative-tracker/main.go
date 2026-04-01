package main

import (
	"log"
	"net/http"
	"os"

	"dnd-initiative-tracker/internal/tracker"
)

func main() {
	rootPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	handler, err := tracker.NewHandler(rootPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on http://127.0.0.1:8000")
	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		log.Fatal(err)
	}
}
