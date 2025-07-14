package main

import (
	"context"
	"fmt"
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
	tunnels, err := client.ListTunnels(ctx)
	if err != nil {
		log.Fatalf("Error creating tunnel %v\n", err)
	}

	for _, tunnel := range tunnels {
		fmt.Printf("Tunnel ID: %s, Name: %s, Secret:%s \n", tunnel.ID, tunnel.Name, tunnel.Secret)
	}
}
