package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/warjiang/cloudflaretunnel-operator/pkg/cloudflare"
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
	tunnel, bytes, err := client.CreateTunnel(ctx, "my-tunnel")
	if err != nil {
		log.Fatalf("Error creating tunnel %v\n", err)
	}
	fmt.Println(string(bytes))
	fmt.Printf("%+v\n", tunnel)
}
