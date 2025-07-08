package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T, handler http.Handler) (*Client, *http.ServeMux, func()) {
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := httptest.NewServer(mux)

	api, err := cloudflare.NewWithAPIToken("test-api-token")
	assert.NoError(t, err, "Failed to create Cloudflare API client")
	api.BaseURL = server.URL

	client := &Client{
		api:       api,
		accountID: "test-account-id",
	}

	return client, mux, func() {
		server.Close()
	}
}

func TestCreateTunnel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "Expected method 'POST'")
		w.Header().Set("Content-Type", "application/json")
		// Use a fixed, valid timestamp to avoid parsing issues.
		// Also, omitting deleted_at as it can cause issues if empty.
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"test-tunnel-id","name":"test-tunnel","created_at":"2024-01-01T00:00:00Z"}}`)
	})

	client, _, teardown := setupTest(t, handler)
	defer teardown()

	tunnel, secret, err := client.CreateTunnel(context.Background(), "test-tunnel")
	assert.NoError(t, err)
	assert.NotNil(t, tunnel)
	assert.NotNil(t, secret)
	assert.Equal(t, "test-tunnel-id", tunnel.ID)
	assert.Equal(t, "test-tunnel", tunnel.Name)
}

func TestGetTunnelByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Expected method 'GET'")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[{"id":"test-tunnel-id","name":"test-tunnel","created_at":"2024-01-01T00:00:00Z"}]}`)
	})

	client, _, teardown := setupTest(t, handler)
	defer teardown()

	tunnel, err := client.GetTunnelByName(context.Background(), "test-tunnel")
	assert.NoError(t, err)
	assert.NotNil(t, tunnel)
	assert.Equal(t, "test-tunnel-id", tunnel.ID)
}

func TestGetTunnelByName_NotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[]}`)
	})

	client, _, teardown := setupTest(t, handler)
	defer teardown()

	_, err := client.GetTunnelByName(context.Background(), "not-found-tunnel")
	assert.Error(t, err)
}

func TestDeleteTunnel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "Expected method 'DELETE'")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"test-tunnel-id"}}`)
	})

	client, _, teardown := setupTest(t, handler)
	defer teardown()

	err := client.DeleteTunnel(context.Background(), "test-tunnel-id")
	assert.NoError(t, err)
}
