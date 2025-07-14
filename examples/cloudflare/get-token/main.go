package main

import (
	"context"
	"fmt"
	"os"

	pkgcloudflare "github.com/warjiang/cloudflare-tunnel-operator/pkg/cloudflare"
)

func main() {
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	apiEmail := os.Getenv("CLOUDFLARE_API_EMAIL")
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	client, err := pkgcloudflare.NewClient(
		accountID,
		pkgcloudflare.WithAPIEmail(apiEmail),
		pkgcloudflare.WithAPIKey(apiKey),
	)
	if err != nil {
		fmt.Println("failed to create cloudflare client:", err)
		return
	}

	tunnels, err := client.ListTunnels(context.Background())
	if err != nil {
		fmt.Println("failed to list tunnels:", err)
		return
	}

	if len(tunnels) == 0 {
		fmt.Println("no tunnels found")
		return
	}

	token, err := client.GetTunnelTokenByID(context.Background(), tunnels[0].ID)
	if err != nil {
		fmt.Println("failed to get tunnel token:", err)
		return
	}

	fmt.Println("Tunnel token:", token)
}