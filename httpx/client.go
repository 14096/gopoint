package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Doer is the minimal interface resource services depend on.
type Doer interface {
	Do(ctx context.Context, req *Request) (*Response, error)
}

// Config configures a Client.
type Config struct {
	// BaseURL is prepended to relative request paths.
	BaseURL string
	// HTTPClient defaults to a client with a 100s timeout.
	HTTPClient *http.Client
	// Token returns a bearer token; nil disables authentication.
	Token func(ctx context.Context) (string, error)
	// Prepare runs on every outgoing request (unless NoPrepare), e.g. to add
	// the SharePoint request digest.
	Prepare func(ctx context.Context, r *http.Request) error
	// Header holds default headers; request headers override them.
	Header    http.Header
	UserAgent string
	Logger    *slog.Logger
	// MaxAttempts is the total number of tries (default 4).
	MaxAttempts int
	// MinBackoff/MaxBackoff bound the exponential backoff (default 500ms/30s).
	MinBackoff time.Duration
	MaxBackoff time.Duration
	// ShouldRetry overrides the default retry decision (429/503/504 and
	// transient network errors on idempotent methods). It receives the
	// default decision; resp is nil when err is set. MaxAttempts still applies.
	ShouldRetry func(req *Request, resp *Response, err error, def bool) bool
	// Middleware wraps Do, outermost first. It sees the whole call, retries
	// included; use a custom HTTPClient.Transport for per-attempt hooks.
	Middleware []Middleware
}

// DoFunc is the signature of Client.Do.
type DoFunc func(ctx context.Context, req *Request) (*Response, error)

// Middleware decorates a DoFunc, e.g. for metrics, tracing or custom logging.
type Middleware func(next DoFunc) DoFunc

// Client is the generic REST client.
type Client struct {
	cfg   Config
	base  *url.URL
	sleep func(context.Context, time.Duration) error
	do    DoFunc
}

var _ Doer = (*Client)(nil)

// New creates a Client. It returns an error if BaseURL is not a valid URL.
func New(cfg Config) (*Client, error) {
	base, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("httpx: invalid BaseURL: %w", err)
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 100 * time.Second}
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 4
	}
	if cfg.MinBackoff <= 0 {
		cfg.MinBackoff = 500 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "github.com/14096/gopoint"
	}
	if cfg.Header.Get("Accept") == "" {
		cfg.Header = cfg.Header.Clone()
		if cfg.Header == nil {
			cfg.Header = http.Header{}
		}
		cfg.Header.Set("Accept", "application/json")
	}
	c := &Client{cfg: cfg, base: base, sleep: sleepCtx}
	c.do = c.send
	for i := len(cfg.Middleware) - 1; i >= 0; i-- {
		c.do = cfg.Middleware[i](c.do)
	}
	return c, nil
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string { return c.base.String() }

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) resolve(r *Request) (string, error) {
	var u *url.URL
	if strings.HasPrefix(r.Path, "http://") || strings.HasPrefix(r.Path, "https://") {
		var err error
		if u, err = url.Parse(r.Path); err != nil {
			return "", err
		}
		if !r.NoAuth && c.cfg.Token != nil && !strings.EqualFold(u.Host, c.base.Host) {
			return "", fmt.Errorf("httpx: refusing to send credentials to foreign host %q", u.Host)
		}
	} else {
		p := r.Path
		if p != "" && !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		var err error
		if u, err = url.Parse(c.base.String() + p); err != nil {
			return "", err
		}
	}
	if len(r.Query) > 0 {
		q := u.Query()
		maps.Copy(q, r.Query)
		u.RawQuery = encodeQuery(q)
	}
	return u.String(), nil
}

// encodeQuery encodes like url.Values.Encode but keeps '$' and ',' readable.
func encodeQuery(q url.Values) string {
	return queryUnescaper.Replace(q.Encode())
}

var queryUnescaper = strings.NewReplacer("%24", "$", "%2C", ",")

// Do sends the request, applying middleware, auth, retries and error mapping.
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	return c.do(ctx, req)
}

func (c *Client) send(ctx context.Context, req *Request) (*Response, error) {
	if req.err != nil {
		return nil, req.err
	}
	target, err := c.resolve(req)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 1; ; attempt++ {
		resp, err := c.once(ctx, req, target)
		var wait time.Duration
		retry := false
		switch {
		case err != nil:
			var ne net.Error
			retry = ctx.Err() == nil && errors.As(err, &ne) && idempotent(req.Method)
			lastErr = err
		case resp.Status >= 200 && resp.Status < 300:
			return resp, nil
		default:
			lastErr = parseError(resp.Status, resp.Header, resp.Body)
			switch resp.Status {
			case http.StatusTooManyRequests, http.StatusServiceUnavailable:
				retry = true
				wait = retryAfter(resp.Header)
			case http.StatusGatewayTimeout:
				retry = idempotent(req.Method)
				wait = retryAfter(resp.Header)
			}
		}
		if c.cfg.ShouldRetry != nil {
			retry = c.cfg.ShouldRetry(req, resp, err, retry)
		}
		if !retry || attempt >= c.cfg.MaxAttempts {
			return nil, lastErr
		}
		if wait <= 0 {
			wait = c.backoff(attempt)
		}
		if c.cfg.Logger != nil {
			c.cfg.Logger.Debug("httpx retry", "attempt", attempt, "wait", wait, "err", lastErr)
		}
		if err := c.sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

func (c *Client) once(ctx context.Context, req *Request, target string) (*Response, error) {
	var body io.Reader
	if req.Body != nil {
		body = bytes.NewReader(req.Body)
	}
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	hr, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	hr.Header.Set("User-Agent", c.cfg.UserAgent)
	for k, vs := range c.cfg.Header {
		hr.Header[http.CanonicalHeaderKey(k)] = slices.Clone(vs)
	}
	for k, vs := range req.Header {
		hr.Header[http.CanonicalHeaderKey(k)] = slices.Clone(vs)
	}
	if !req.NoAuth && c.cfg.Token != nil {
		tok, err := c.cfg.Token(ctx)
		if err != nil {
			return nil, fmt.Errorf("httpx: acquire token: %w", err)
		}
		hr.Header.Set("Authorization", "Bearer "+tok)
	}
	if !req.NoPrepare && c.cfg.Prepare != nil {
		if err := c.cfg.Prepare(ctx, hr); err != nil {
			return nil, err
		}
	}
	res, err := c.cfg.HTTPClient.Do(hr)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return &Response{Status: res.StatusCode, Header: res.Header, Body: b}, nil
}

func idempotent(m string) bool {
	switch m {
	case "", http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	}
	return false
}

func retryAfter(h http.Header) time.Duration {
	v := h.Get("Retry-After")
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func (c *Client) backoff(attempt int) time.Duration {
	d := c.cfg.MinBackoff << (attempt - 1)
	if d <= 0 || d > c.cfg.MaxBackoff {
		d = c.cfg.MaxBackoff
	}
	return d/2 + time.Duration(rand.Int64N(int64(d/2)+1))
}

// JSON sends req through d and decodes the response into out (may be nil).
func JSON(ctx context.Context, d Doer, req *Request, out any) error {
	resp, err := d.Do(ctx, req)
	if err != nil {
		return err
	}
	return resp.JSON(out)
}

// Call is a generic helper: body (if not nil) is JSON encoded, the response is
// decoded into out (if not nil). Options adjust this call only.
func (c *Client) Call(ctx context.Context, method, path string, body, out any, opts ...Option) error {
	_, err := c.Send(ctx, method, path, body, out, opts...)
	return err
}

// Send is like Call but also returns the response, giving access to the
// status, headers and ETag.
func (c *Client) Send(ctx context.Context, method, path string, body, out any, opts ...Option) (*Response, error) {
	r := NewRequest(method, path)
	if body != nil {
		r.WithJSON(body)
	}
	r.Apply(opts...)
	resp, err := c.Do(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp, resp.JSON(out)
}

// Get sends a GET request and decodes the response into out.
func (c *Client) Get(ctx context.Context, path string, out any, opts ...Option) error {
	return c.Call(ctx, http.MethodGet, path, nil, out, opts...)
}

// Post sends body as JSON in a POST request and decodes the response into out.
func (c *Client) Post(ctx context.Context, path string, body, out any, opts ...Option) error {
	return c.Call(ctx, http.MethodPost, path, body, out, opts...)
}

// Patch sends body as JSON in a PATCH request and decodes the response into out.
func (c *Client) Patch(ctx context.Context, path string, body, out any, opts ...Option) error {
	return c.Call(ctx, http.MethodPatch, path, body, out, opts...)
}

// Put sends body as JSON in a PUT request and decodes the response into out.
func (c *Client) Put(ctx context.Context, path string, body, out any, opts ...Option) error {
	return c.Call(ctx, http.MethodPut, path, body, out, opts...)
}

// Delete sends a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, opts ...Option) error {
	return c.Call(ctx, http.MethodDelete, path, nil, nil, opts...)
}

// DoHTTP sends a standard library request through the client (auth, retries,
// error mapping). The request URL may be absolute or, if it has no host,
// relative to BaseURL. Its body is read fully.
func (c *Client) DoHTTP(ctx context.Context, r *http.Request) (*Response, error) {
	req, err := NewRequestFromHTTP(r)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Call sends a request through d and returns the decoded response as T.
func Call[T any](ctx context.Context, d Doer, method, path string, body any, opts ...Option) (T, error) {
	var out T
	r := NewRequest(method, path)
	if body != nil {
		r.WithJSON(body)
	}
	r.Apply(opts...)
	err := JSON(ctx, d, r, &out)
	return out, err
}

// Get sends a GET request through d and returns the decoded response as T.
func Get[T any](ctx context.Context, d Doer, path string, opts ...Option) (T, error) {
	return Call[T](ctx, d, http.MethodGet, path, nil, opts...)
}

// Post sends body as JSON through d and returns the decoded response as T.
func Post[T any](ctx context.Context, d Doer, path string, body any, opts ...Option) (T, error) {
	return Call[T](ctx, d, http.MethodPost, path, body, opts...)
}
