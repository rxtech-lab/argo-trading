package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sirily11/argo-trading-go/cmd/download/clients"
)

func main() {

	client := clients.NewPolygonClient()

	path, err := client.Download("APPL", "data/GOOG.csv", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 2, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		log.Fatalf("Failed to download data: %v", err)
	}

	fmt.Printf("Downloaded data to %s\n", path)
}
