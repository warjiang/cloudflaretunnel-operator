package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/warjiang/cloudflare-tunnel-operator/pkg/cloudflare"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiToken := os.Getenv("API_TOKEN")
	accountID := os.Getenv("ACCOUNT_ID")
	client, err := cloudflare.NewClient(
		accountID,
		cloudflare.WithAPIToken(apiToken),
	)
	if err != nil {
		log.Fatal("Error creating client")
	}

	ctx := context.TODO()
	err = client.DeleteTunnelByName(ctx, "my-tunnel")
	if err != nil {
		log.Fatalf("Error deleted tunnel %v\n", err)
	}
}
