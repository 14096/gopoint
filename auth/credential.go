package auth

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// DefaultAuthority is the public-cloud Entra ID login host.
const DefaultAuthority = "https://login.microsoftonline.com"

// expiryMargin is how long before expiry a cached token is considered stale.
const expiryMargin = 5 * time.Minute

// AccessToken is a bearer token with its expiry.
type AccessToken struct {
	Token     string
	ExpiresOn time.Time
}

// TokenOptions selects what the token is for. Scopes are OAuth2 v2 scopes,
// typically "<resource>/.default".
type TokenOptions struct {
	Scopes []string
}

// Credential returns access tokens. Implementations are safe for concurrent use.
type Credential interface {
	Token(ctx context.Context, opts TokenOptions) (AccessToken, error)
}

// Options are settings common to all credentials.
type Options struct {
	// Authority overrides the login host (sovereign clouds, tests).
	Authority string
	// HTTPClient is used for calls to the identity platform.
	HTTPClient *http.Client
}

func (o Options) authority() string {
	if o.Authority == "" {
		return DefaultAuthority
	}
	return strings.TrimRight(o.Authority, "/")
}

func (o Options) client() *http.Client {
	if o.HTTPClient == nil {
		return &http.Client{Timeout: 30 * time.Second}
	}
	return o.HTTPClient
}

// tokenCache caches tokens per scope set. Fetches are serialised so that
// concurrent callers never trigger duplicate (possibly interactive) flows.
type tokenCache struct {
	mu  sync.Mutex
	m   map[string]AccessToken
	now func() time.Time
}

func scopeKey(scopes []string) string {
	return strings.Join(slices.Sorted(slices.Values(scopes)), " ")
}

func (c *tokenCache) get(key string, fetch func() (AccessToken, error)) (AccessToken, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now
	if c.now != nil {
		now = c.now
	}
	if t, ok := c.m[key]; ok && now().Add(expiryMargin).Before(t.ExpiresOn) {
		return t, nil
	}
	t, err := fetch()
	if err != nil {
		return AccessToken{}, err
	}
	if c.m == nil {
		c.m = map[string]AccessToken{}
	}
	c.m[key] = t
	return t, nil
}
