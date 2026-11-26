package graph

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/graph/columns"
	"github.com/14096/gopoint/graph/contenttypes"
	"github.com/14096/gopoint/graph/drives"
	"github.com/14096/gopoint/graph/items"
	"github.com/14096/gopoint/graph/lists"
	"github.com/14096/gopoint/graph/sites"
	"github.com/14096/gopoint/httpx"
)

// DefaultBaseURL is the public-cloud Graph v1.0 endpoint.
const DefaultBaseURL = "https://graph.microsoft.com/v1.0"

// Options tune the client; all fields are optional.
type Options struct {
	// BaseURL overrides the endpoint (national clouds, beta API, tests). The
	// token scope is derived from its host: https://<host>/.default.
	BaseURL     string
	HTTPClient  *http.Client
	Logger      *slog.Logger
	UserAgent   string
	MaxAttempts int
}

// Client talks to Microsoft Graph.
type Client struct {
	raw *httpx.Client
}

// New creates a Graph client.
func New(cred auth.Credential, o Options) (*Client, error) {
	if cred == nil {
		return nil, errors.New("graph: credential is required")
	}
	if o.BaseURL == "" {
		o.BaseURL = DefaultBaseURL
	}
	u, err := url.Parse(o.BaseURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("graph: invalid base URL %q", o.BaseURL)
	}
	scope := "https://" + u.Host + "/.default"
	raw, err := httpx.New(httpx.Config{
		BaseURL: o.BaseURL, HTTPClient: o.HTTPClient, Logger: o.Logger,
		UserAgent: o.UserAgent, MaxAttempts: o.MaxAttempts,
		Token: func(ctx context.Context) (string, error) {
			t, err := cred.Token(ctx, auth.TokenOptions{Scopes: []string{scope}})
			return t.Token, err
		},
	})
	if err != nil {
		return nil, err
	}
	return &Client{raw: raw}, nil
}

// Raw returns the generic client for endpoints that have no typed wrapper yet.
func (c *Client) Raw() *httpx.Client { return c.raw }

// Resource services. Add new resources here; each only needs an httpx.Doer.

// Sites returns the sites service.
func (c *Client) Sites() *sites.Service { return sites.New(c.raw) }

// Lists returns the lists service.
func (c *Client) Lists() *lists.Service { return lists.New(c.raw) }

// Columns returns the columns service.
func (c *Client) Columns() *columns.Service { return columns.New(c.raw) }

// Items returns the drive items service.
func (c *Client) Items() *items.Service { return items.New(c.raw) }

// ContentTypes returns the content types service.
func (c *Client) ContentTypes() *contenttypes.Service { return contenttypes.New(c.raw) }

// Drives returns the drives service.
func (c *Client) Drives() *drives.Service { return drives.New(c.raw) }
