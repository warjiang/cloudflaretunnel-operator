package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudflare/cloudflare-go/v3"
	"github.com/cloudflare/cloudflare-go/v3/option"
	"github.com/cloudflare/cloudflare-go/v3/zero_trust"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiToken := os.Getenv("API_TOKEN")
	accountID := os.Getenv("ACCOUNT_ID")
	authEmail := os.Getenv("AUTH_EMAIL")
	authKey := os.Getenv("AUTH_KEY")
	client := cloudflare.NewClient(
		option.WithAPIToken(apiToken),
		option.WithAPIEmail(authEmail),
		option.WithAPIKey(authKey),
	)
	page, err := client.ZeroTrust.Tunnels.List(context.TODO(), zero_trust.TunnelListParams{
		AccountID: cloudflare.F(accountID),
	})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("%+v\n", page)
}
