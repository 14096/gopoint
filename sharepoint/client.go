package sharepoint

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/contenttypes"
	"github.com/14096/gopoint/sharepoint/fields"
	"github.com/14096/gopoint/sharepoint/files"
	"github.com/14096/gopoint/sharepoint/items"
	"github.com/14096/gopoint/sharepoint/lists"
	"github.com/14096/gopoint/sharepoint/web"
)

// Options tune the client; all fields are optional.
type Options struct {
	HTTPClient  *http.Client
	Logger      *slog.Logger
	UserAgent   string
	MaxAttempts int
}

// Client talks to the SharePoint REST API of one site (or web).
type Client struct {
	siteURL string
	raw     *httpx.Client
	digest  digestCache
}

// New creates a client for siteURL (e.g. https://contoso.sharepoint.com/sites/team).
// Tokens are requested for the tenant's SharePoint host with the ".default" scope.
func New(siteURL string, cred auth.Credential, o Options) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(siteURL, "/"))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("sharepoint: invalid site URL %q", siteURL)
	}
	if cred == nil {
		return nil, errors.New("sharepoint: credential is required")
	}
	scope := "https://" + u.Host + "/.default"
	c := &Client{siteURL: u.String()}
	raw, err := httpx.New(httpx.Config{
		BaseURL:     c.siteURL + "/_api",
		HTTPClient:  o.HTTPClient,
		Logger:      o.Logger,
		UserAgent:   o.UserAgent,
		MaxAttempts: o.MaxAttempts,
		Header:      http.Header{"Accept": {"application/json;odata=nometadata"}},
		Token: func(ctx context.Context) (string, error) {
			t, err := cred.Token(ctx, auth.TokenOptions{Scopes: []string{scope}})
			return t.Token, err
		},
		Prepare: c.addDigest,
	})
	if err != nil {
		return nil, err
	}
	c.raw = raw
	return c, nil
}

// SiteURL returns the site this client is scoped to.
func (c *Client) SiteURL() string { return c.siteURL }

// Raw returns the generic client (base URL <site>/_api) for endpoints that have
// no typed wrapper yet.
func (c *Client) Raw() *httpx.Client { return c.raw }

// Resource services. Add new resources here; each only needs an httpx.Doer.

// Web returns the web service.
func (c *Client) Web() *web.Service { return web.New(c.raw) }

// Lists returns the lists service.
func (c *Client) Lists() *lists.Service { return lists.New(c.raw) }

// Fields returns the fields service.
func (c *Client) Fields() *fields.Service { return fields.New(c.raw) }

// ContentTypes returns the content types service.
func (c *Client) ContentTypes() *contenttypes.Service { return contenttypes.New(c.raw) }

// Items returns the list items service.
func (c *Client) Items() *items.Service { return items.New(c.raw) }

// Files returns the files service.
func (c *Client) Files() *files.Service { return files.New(c.raw) }

type digestCache struct {
	mu      sync.Mutex
	value   string
	expires time.Time
}

// addDigest attaches X-RequestDigest to state-changing requests.
func (c *Client) addDigest(ctx context.Context, r *http.Request) error {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return nil
	}
	if r.Header.Get("X-RequestDigest") != "" {
		return nil
	}
	d, err := c.requestDigest(ctx)
	if err != nil {
		return err
	}
	r.Header.Set("X-RequestDigest", d)
	return nil
}

func (c *Client) requestDigest(ctx context.Context) (string, error) {
	c.digest.mu.Lock()
	defer c.digest.mu.Unlock()
	if c.digest.value != "" && time.Now().Before(c.digest.expires) {
		return c.digest.value, nil
	}
	req := httpx.NewRequest(http.MethodPost, "contextinfo")
	req.NoPrepare = true
	var out struct {
		FormDigestValue          string `json:"FormDigestValue"`
		FormDigestTimeoutSeconds int    `json:"FormDigestTimeoutSeconds"`
		D                        *struct {
			Info struct {
				FormDigestValue          string `json:"FormDigestValue"`
				FormDigestTimeoutSeconds int    `json:"FormDigestTimeoutSeconds"`
			} `json:"GetContextWebInformation"`
		} `json:"d"`
	}
	if err := httpx.JSON(ctx, c.raw, req, &out); err != nil {
		return "", fmt.Errorf("sharepoint: request digest: %w", err)
	}
	v, secs := out.FormDigestValue, out.FormDigestTimeoutSeconds
	if out.D != nil {
		v, secs = out.D.Info.FormDigestValue, out.D.Info.FormDigestTimeoutSeconds
	}
	if v == "" {
		return "", errors.New("sharepoint: empty request digest")
	}
	if secs <= 0 {
		secs = 1800
	}
	c.digest.value = v
	c.digest.expires = time.Now().Add(time.Duration(secs)*time.Second - 30*time.Second)
	return v, nil
}
