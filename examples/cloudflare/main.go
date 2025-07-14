package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/warjiang/cloudflare-tunnel-operator/pkg/cloudflare"
	"log"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := os.Getenv("API_KEY")
	accountID := os.Getenv("ACCOUNT_ID")
	client, err := cloudflare.NewClient(apiKey, accountID)
	if err != nil {
		log.Fatal("Error creating client")
	}

	ctx := context.TODO()
	tunnel, bytes, err := client.CreateTunnel(ctx, "my-tunnel")
	if err != nil {
		log.Fatalf("Error creating tunnel +%v\n", err)
	}
	fmt.Println(string(bytes))
	fmt.Printf("%+v\n", tunnel)
}
