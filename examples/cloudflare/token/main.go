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
		cloudflare.WithDebug(false),
	)
	if err != nil {
		log.Fatal("Error creating client")
	}

	ctx := context.TODO()
	tunnel, err := client.GetTunnelTokenByName(ctx, "my-tunnel")
	if err != nil {
		log.Fatalf("Error creating tunnel %v\n", err)
	}
	fmt.Printf("token is %s\n", tunnel)
	fmt.Println("you can run the following command to start the tunnel:")
	// nolint:lll
	fmt.Printf("docker run docker.cr.20220625.xyz/cloudflare/cloudflared:latest tunnel --no-autoupdate run --token %s\n", tunnel)
}
