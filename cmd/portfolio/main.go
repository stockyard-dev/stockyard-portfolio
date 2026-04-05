package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-portfolio/internal/server"
	"github.com/stockyard-dev/stockyard-portfolio/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9808"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./portfolio-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("portfolio: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Portfolio — Self-hosted portfolio and gallery management\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Questions? hello@stockyard.dev — I read every message\n\n", port, port)
	log.Printf("portfolio: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
