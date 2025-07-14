package cloudflare

import "net/http"

// clientOptions holds the configuration for the Cloudflare client.
type clientOptions struct {
	apiToken   string
	accountID  string
	email      string
	globalKey  string
	debug      bool
	httpClient *http.Client
	baseURL    string
}

// Option is a function that configures the client.
type Option func(*clientOptions)

// WithAPIToken sets the API token.
func WithAPIToken(token string) Option {
	return func(o *clientOptions) {
		o.apiToken = token
	}
}

// WithAccountID sets the account ID.
func WithAccountID(id string) Option {
	return func(o *clientOptions) {
		o.accountID = id
	}
}

// WithEmail sets the email for authentication.
func WithEmail(email string) Option {
	return func(o *clientOptions) {
		o.email = email
	}
}

// WithGlobalKey sets the global API key for authentication.
func WithGlobalKey(key string) Option {
	return func(o *clientOptions) {
		o.globalKey = key
	}
}

// WithDebug enables debug mode.
func WithDebug(debug bool) Option {
	return func(o *clientOptions) {
		o.debug = debug
	}
}

// WithHTTPClient sets a custom http client for tests.
func WithHTTPClient(client *http.Client) Option {
	return func(o *clientOptions) {
		o.httpClient = client
	}
}

// WithBaseURL sets a custom base URL for tests.
func WithBaseURL(url string) Option {
	return func(o *clientOptions) {
		o.baseURL = url
	}
}
