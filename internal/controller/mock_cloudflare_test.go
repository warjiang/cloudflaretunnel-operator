package controller

import (
	"context"

	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/mock"
	pkgcloudflare "github.com/warjiang/cloudflare-tunnel-operator/pkg/cloudflare"
)

type MockCloudflareClient struct {
	mock.Mock
}

func (m *MockCloudflareClient) CreateTunnel(ctx context.Context, name string) (*cloudflare.Tunnel, []byte, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Get(1).([]byte), args.Error(2)
	}
	return args.Get(0).(*cloudflare.Tunnel), args.Get(1).([]byte), args.Error(2)
}

func (m *MockCloudflareClient) GetTunnelByName(ctx context.Context, name string) (*cloudflare.Tunnel, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cloudflare.Tunnel), args.Error(1)
}

func (m *MockCloudflareClient) DeleteTunnel(ctx context.Context, tunnelID string) error {
	args := m.Called(ctx, tunnelID)
	return args.Error(0)
}

var _ pkgcloudflare.ClientInterface = &MockCloudflareClient{}
