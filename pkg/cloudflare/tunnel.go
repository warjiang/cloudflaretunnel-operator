package cloudflare

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

// ClientInterface defines the interface for the Cloudflare client.
type ClientInterface interface {
	CreateTunnel(ctx context.Context, name string) (*cloudflare.Tunnel, []byte, error)
	GetTunnelByName(ctx context.Context, name string) (*cloudflare.Tunnel, error)
	DeleteTunnelByID(ctx context.Context, tunnelID string) error
	DeleteTunnelByName(ctx context.Context, name string) error
	ListTunnels(ctx context.Context) ([]cloudflare.Tunnel, error)
	GetTunnelTokenByID(ctx context.Context, tunnelID string) (string, error)
	GetTunnelTokenByName(ctx context.Context, name string) (string, error)
	UpsertTunnelConfiguration(ctx context.Context, tunnelID string, rules []TunnelIngressRule) error
	EnsureCNAMERecord(ctx context.Context, zoneID, hostname, target string) (string, error)
	DeleteDNSRecordByID(ctx context.Context, zoneID, recordID string) error
}

// TunnelIngressRule is a high-level tunnel ingress entry managed by the operator.
type TunnelIngressRule struct {
	Hostname string
	Path     string
	Service  string
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

// DeleteTunnelByID deletes a Cloudflare tunnel.
func (c *Client) DeleteTunnelByID(ctx context.Context, tunnelID string) error {
	rc := cloudflare.AccountIdentifier(c.accountID)
	err := c.api.DeleteTunnel(ctx, rc, tunnelID)
	if err != nil {
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}
	return nil
}

// DeleteTunnelByName deletes a Cloudflare tunnel by name.
func (c *Client) DeleteTunnelByName(ctx context.Context, name string) error {
	tunnel, err := c.GetTunnelByName(ctx, name)
	if err != nil {
		return err
	}
	return c.DeleteTunnelByID(ctx, tunnel.ID)
}

// ListTunnels lists all Cloudflare tunnels.
func (c *Client) ListTunnels(ctx context.Context) ([]cloudflare.Tunnel, error) {
	rc := cloudflare.AccountIdentifier(c.accountID)
	tunnels, _, err := c.api.ListTunnels(ctx, rc, cloudflare.TunnelListParams{
		IsDeleted: cloudflare.BoolPtr(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tunnels: %w", err)
	}

	return tunnels, nil
}

// GetTunnelTokenByID gets a Cloudflare tunnel token by ID.
func (c *Client) GetTunnelTokenByID(ctx context.Context, tunnelID string) (string, error) {
	rc := cloudflare.AccountIdentifier(c.accountID)
	token, err := c.api.GetTunnelToken(ctx, rc, tunnelID)
	if err != nil {
		return "", fmt.Errorf("failed to get tunnel token: %w", err)
	}
	return token, nil
}

// GetTunnelTokenByName gets a Cloudflare tunnel token by name.
func (c *Client) GetTunnelTokenByName(ctx context.Context, name string) (string, error) {
	tunnel, err := c.GetTunnelByName(ctx, name)
	if err != nil {
		return "", err
	}
	return c.GetTunnelTokenByID(ctx, tunnel.ID)
}

// UpsertTunnelConfiguration creates or updates the remotely-managed tunnel ingress configuration.
func (c *Client) UpsertTunnelConfiguration(ctx context.Context, tunnelID string, rules []TunnelIngressRule) error {
	if tunnelID == "" {
		return fmt.Errorf("tunnelID is required")
	}
	if len(rules) == 0 {
		return fmt.Errorf("at least one ingress rule is required")
	}

	ingress := make([]cloudflare.UnvalidatedIngressRule, 0, len(rules))
	for _, rule := range rules {
		ingress = append(ingress, cloudflare.UnvalidatedIngressRule{
			Hostname: rule.Hostname,
			Path:     rule.Path,
			Service:  rule.Service,
		})
	}

	rc := cloudflare.AccountIdentifier(c.accountID)
	_, err := c.api.UpdateTunnelConfiguration(ctx, rc, cloudflare.TunnelConfigurationParams{
		TunnelID: tunnelID,
		Config: cloudflare.TunnelConfiguration{
			Ingress: ingress,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update tunnel configuration: %w", err)
	}
	return nil
}

// EnsureCNAMERecord creates or updates a CNAME record for a hostname.
func (c *Client) EnsureCNAMERecord(ctx context.Context, zoneID, hostname, target string) (string, error) {
	if zoneID == "" {
		return "", fmt.Errorf("zoneID is required")
	}
	if hostname == "" {
		return "", fmt.Errorf("hostname is required")
	}
	if target == "" {
		return "", fmt.Errorf("target is required")
	}

	normalizedTarget := strings.TrimSuffix(target, ".")
	proxied := true
	rc := cloudflare.ZoneIdentifier(zoneID)
	records, _, err := c.api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Type: "CNAME",
		Name: hostname,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list dns records: %w", err)
	}

	for _, record := range records {
		if strings.EqualFold(record.Name, hostname) && strings.EqualFold(record.Type, "CNAME") {
			if strings.EqualFold(strings.TrimSuffix(record.Content, "."), normalizedTarget) &&
				record.Proxied != nil &&
				*record.Proxied {
				return record.ID, nil
			}
			updated, updateErr := c.api.UpdateDNSRecord(ctx, rc, cloudflare.UpdateDNSRecordParams{
				ID:      record.ID,
				Type:    "CNAME",
				Name:    hostname,
				Content: normalizedTarget,
				Proxied: &proxied,
				TTL:     1,
			})
			if updateErr != nil {
				return "", fmt.Errorf("failed to update dns record: %w", updateErr)
			}
			return updated.ID, nil
		}
	}

	created, err := c.api.CreateDNSRecord(ctx, rc, cloudflare.CreateDNSRecordParams{
		Type:    "CNAME",
		Name:    hostname,
		Content: normalizedTarget,
		Proxied: &proxied,
		TTL:     1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create dns record: %w", err)
	}
	return created.ID, nil
}

// DeleteDNSRecordByID deletes a DNS record by zone and record ID.
func (c *Client) DeleteDNSRecordByID(ctx context.Context, zoneID, recordID string) error {
	if zoneID == "" || recordID == "" {
		return nil
	}
	if err := c.api.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), recordID); err != nil {
		return fmt.Errorf("failed to delete dns record: %w", err)
	}
	return nil
}
