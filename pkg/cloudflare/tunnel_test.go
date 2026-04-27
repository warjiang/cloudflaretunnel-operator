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

func TestDeleteTunnelByID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "Expected method 'DELETE'")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"test-tunnel-id"}}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	err := client.DeleteTunnelByID(context.Background(), "test-tunnel-id")
	assert.NoError(t, err)
}

func TestDeleteTunnelByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/accounts/mock_account_id/cfd_tunnel":
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
		case "/accounts/mock_account_id/cfd_tunnel/test-tunnel-id":
			_, _ = fmt.Fprint(w, `{"success":true,"errors":[],"messages":[],"result":{"id":"test-tunnel-id"}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	err := client.DeleteTunnelByName(context.Background(), "test-tunnel")
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

func TestGetTunnelTokenByID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Expected method 'GET'")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"success":true,
			"errors":[],
			"messages":[],
			"result":"test-token"
		}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	token, err := client.GetTunnelTokenByID(context.Background(), "test-tunnel-id")
	assert.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestGetTunnelTokenByName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/accounts/mock_account_id/cfd_tunnel":
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
		case "/accounts/mock_account_id/cfd_tunnel/test-tunnel-id/token":
			_, _ = fmt.Fprint(w, `{
				"success":true,
				"errors":[],
				"messages":[],
				"result":"test-token"
			}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	token, err := client.GetTunnelTokenByName(context.Background(), "test-tunnel")
	assert.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestUpsertTunnelConfiguration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method, "Expected method 'PUT'")
		assert.Equal(t, "/accounts/mock_account_id/cfd_tunnel/test-tunnel-id/configurations", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"tunnel_id": "test-tunnel-id",
				"config": {
					"ingress": [
						{
							"hostname": "app.example.com",
							"service": "http://svc.default.svc.cluster.local:8080"
						}
					]
				}
			}
		}`)
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	err := client.UpsertTunnelConfiguration(context.Background(), "test-tunnel-id", []TunnelIngressRule{
		{
			Hostname: "app.example.com",
			Service:  "http://svc.default.svc.cluster.local:8080",
		},
	})
	assert.NoError(t, err)
}

func TestEnsureCNAMERecordCreate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-id/dns_records":
			_, _ = fmt.Fprint(w, `{
				"success": true,
				"errors": [],
				"messages": [],
				"result": [],
				"result_info": {
					"page": 1,
					"per_page": 100,
					"count": 0,
					"total_count": 0
				}
			}`)
		case r.Method == http.MethodPost && r.URL.Path == "/zones/zone-id/dns_records":
			_, _ = fmt.Fprint(w, `{
				"success": true,
				"errors": [],
				"messages": [],
				"result": {
					"id": "dns-record-id",
					"type": "CNAME",
					"name": "app.example.com",
					"content": "test-tunnel-id.cfargotunnel.com",
					"proxied": true
				}
			}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client, teardown := setupTest(t, handler)
	defer teardown()

	recordID, err := client.EnsureCNAMERecord(
		context.Background(),
		"zone-id",
		"app.example.com",
		"test-tunnel-id.cfargotunnel.com",
	)
	assert.NoError(t, err)
	assert.Equal(t, "dns-record-id", recordID)
}
