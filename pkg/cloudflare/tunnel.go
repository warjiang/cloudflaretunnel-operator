package cloudflare

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// ClientInterface defines the interface for the Cloudflare client.
type ClientInterface interface {
	CreateTunnel(ctx context.Context, name string) (*cloudflare.Tunnel, []byte, error)
	GetTunnelByName(ctx context.Context, name string) (*cloudflare.Tunnel, error)
	DeleteTunnel(ctx context.Context, tunnelID string) error
	ListTunnels(ctx context.Context) ([]cloudflare.Tunnel, error)
}

// Client is a wrapper around the Cloudflare API client.
type Client struct {
	api       *cloudflare.API
	accountID string
}

var _ ClientInterface = &Client{}

// NewClient creates a new Cloudflare client.
func NewClient(accountID string, opts ...Option) (*Client, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID is required")
	}

	// apply options
	cfg := &clientOptions{}
	for _, opt := range opts {
		opt(cfg)
	}

	var api *cloudflare.API
	var err error

	// convert options to official cloudflare options
	var cloudflareOpts []cloudflare.Option
	if cfg.httpClient != nil {
		cloudflareOpts = append(cloudflareOpts, cloudflare.HTTPClient(cfg.httpClient))
	}
	if cfg.baseURL != "" {
		cloudflareOpts = append(cloudflareOpts, cloudflare.BaseURL(cfg.baseURL))
	}
	cloudflareOpts = append(cloudflareOpts, cloudflare.Debug(cfg.debug))

	if cfg.apiToken != "" {
		api, err = cloudflare.NewWithAPIToken(cfg.apiToken, cloudflareOpts...)
	} else if cfg.globalKey != "" && cfg.email != "" {
		api, err = cloudflare.New(cfg.globalKey, cfg.email, cloudflareOpts...)
	} else {
		return nil, fmt.Errorf("either API token or global key and email must be provided")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare client: %w", err)
	}

	return &Client{
		api:       api,
		accountID: accountID,
	}, nil
}

// CreateTunnel creates a new Cloudflare tunnel.
func (c *Client) CreateTunnel(ctx context.Context, name string) (*cloudflare.Tunnel, []byte, error) {
	rc := cloudflare.AccountIdentifier(c.accountID)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, nil, fmt.Errorf("failed to generate tunnel secret: %w", err)
	}

	tunnel, err := c.api.CreateTunnel(ctx, rc, cloudflare.TunnelCreateParams{
		Name:   name,
		Secret: base64.StdEncoding.EncodeToString(secretBytes),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tunnel: %w", err)
	}
	return &tunnel, secretBytes, nil
}

// GetTunnelByName finds a tunnel by its name.
func (c *Client) GetTunnelByName(ctx context.Context, name string) (*cloudflare.Tunnel, error) {
	tunnels, err := c.ListTunnels(ctx)
	if err != nil {
		return nil, err
	}

	for _, tunnel := range tunnels {
		if tunnel.Name == name {
			t := tunnel
			return &t, nil
		}
	}

	return nil, fmt.Errorf("tunnel with name '%s' not found", name)
}

// DeleteTunnel deletes a Cloudflare tunnel.
func (c *Client) DeleteTunnel(ctx context.Context, tunnelID string) error {
	rc := cloudflare.AccountIdentifier(c.accountID)
	err := c.api.DeleteTunnel(ctx, rc, tunnelID)
	if err != nil {
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}
	return nil
}

// ListTunnels lists all Cloudflare tunnels.
func (c *Client) ListTunnels(ctx context.Context) ([]cloudflare.Tunnel, error) {
	rc := cloudflare.AccountIdentifier(c.accountID)
	tunnels, _, err := c.api.ListTunnels(ctx, rc, cloudflare.TunnelListParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tunnels: %w", err)
	}

	return tunnels, nil
}
