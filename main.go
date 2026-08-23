package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

// The frontend is built into web/dist before the Go binary is built.
//
//go:embed web/dist
var frontend embed.FS

func main() {
	dist, err := fs.Sub(frontend, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.FS(dist)))

	log.Println("Tsumugi Industry listening on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
