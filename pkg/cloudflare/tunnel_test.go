package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T, handler http.Handler) (*Client, func()) {
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	server := httptest.NewServer(mux)

	client, err := NewClient(
		"mock_account_id",
		WithAPIToken("test-api-token"),
		WithAccountID("test-account-id"),
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	assert.NoError(t, err, "Failed to create Cloudflare API client")

	return client, func() {
		server.Close()
	}
}

func TestCreateTunnel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "Expected method 'POST'")
		w.Header().Set("Content-Type", "application/json")
		// Use a fixed, valid timestamp to avoid parsing issues.
		// Also, omitting deleted_at as it can cause issues if empty.
		_, _ = fmt.Fprint(w, `{
			"success":true,
			"errors":[],
			"messages":[],
			"result":{
				"id":"test-tunnel-id",
				"name":"test-tunnel",
				"created_at":"2024-01-01T00:00:00Z"
			}
		}`)
	})

	client, teardown := setupTest(t, handler)
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
		_, _ = fmt.Fprint(w, `{
			"success":true,
			"errors":[],
			"messages":[],
			"result":[{
				"id":"test-tunnel-id",
				"name":"test-tunnel",
				"created_at":"2024-01-01T00:00:00Z"
			}]
		}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	tunnel, err := client.GetTunnelByName(context.Background(), "test-tunnel")
	assert.NoError(t, err)
	assert.NotNil(t, tunnel)
	assert.Equal(t, "test-tunnel-id", tunnel.ID)
}

func TestGetTunnelByNameNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":[]}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	_, err := client.GetTunnelByName(context.Background(), "not-found-tunnel")
	assert.Error(t, err)
}

func TestDeleteTunnel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "Expected method 'DELETE'")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"test-tunnel-id"}}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	err := client.DeleteTunnel(context.Background(), "test-tunnel-id")
	assert.NoError(t, err)
}

func TestListTunnels(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Expected method 'GET'")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"success":true,
			"errors":[],
			"messages":[],
			"result":[{
				"id":"test-tunnel-id",
				"name":"test-tunnel",
				"created_at":"2024-01-01T00:00:00Z"
			}]
		}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	tunnels, err := client.ListTunnels(context.Background())
	assert.NoError(t, err)
	assert.Len(t, tunnels, 1)
	assert.Equal(t, "test-tunnel-id", tunnels[0].ID)
}
