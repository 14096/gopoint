package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

// BrowserConfig configures an interactive browser credential.
type BrowserConfig struct {
	TenantID string
	ClientID string
	// Open launches the browser at the URL. Defaults to the OS opener.
	Open func(url string) error
	// LoginHint pre-fills the username.
	LoginHint string
	// Timeout bounds the wait for sign-in (default 5 minutes).
	Timeout time.Duration
	Options
}

// InteractiveBrowser is a delegated credential using the authorization code
// flow with PKCE and a loopback redirect (http://localhost:<port>). The app
// registration needs "http://localhost" as a public-client redirect URI.
type InteractiveBrowser struct {
	d   *delegated
	cfg BrowserConfig
}

// NewInteractiveBrowser returns an interactive browser credential.
func NewInteractiveBrowser(cfg BrowserConfig) (*InteractiveBrowser, error) {
	if cfg.TenantID == "" || cfg.ClientID == "" {
		return nil, errors.New("auth: TenantID and ClientID are required")
	}
	if cfg.Open == nil {
		cfg.Open = openBrowser
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Minute
	}
	b := &InteractiveBrowser{cfg: cfg}
	b.d = &delegated{
		opts: cfg.Options, ep: newEndpoints(cfg.authority(), cfg.TenantID),
		clientID: cfg.ClientID, interactive: b.signIn,
	}
	return b, nil
}

// Token implements Credential.
func (b *InteractiveBrowser) Token(ctx context.Context, o TokenOptions) (AccessToken, error) {
	return b.d.Token(ctx, o)
}

type callback struct {
	code string
	err  error
}

func (b *InteractiveBrowser) signIn(ctx context.Context, scope string) (tokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, b.cfg.Timeout)
	defer cancel()

	verifier := rand.Text() + rand.Text()
	state := rand.Text()
	challenge := sha256.Sum256([]byte(verifier))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return tokenResponse{}, err
	}
	redirect := fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)

	result := make(chan callback, 1)
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		var cb callback
		switch {
		case q.Get("state") == "" && q.Get("code") == "" && q.Get("error") == "":
			http.NotFound(w, r) // e.g. favicon
			return
		case subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1:
			cb.err = errors.New("auth: state mismatch in authorization response")
		case q.Get("error") != "":
			cb.err = &OAuthError{Code: q.Get("error"), Description: q.Get("error_description")}
		default:
			cb.code = q.Get("code")
		}
		msg := "Sign-in complete. You can close this window."
		if cb.err != nil {
			w.WriteHeader(http.StatusBadRequest)
			msg = "Sign-in failed: " + html.EscapeString(cb.err.Error())
		}
		fmt.Fprintf(w, "<html><body>%s</body></html>", msg)
		select {
		case result <- cb:
		default:
		}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	q := url.Values{
		"client_id":             {b.cfg.ClientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirect},
		"response_mode":         {"query"},
		"scope":                 {scope},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	if b.cfg.LoginHint != "" {
		q.Set("login_hint", b.cfg.LoginHint)
	}
	if err := b.cfg.Open(b.d.ep.authorize + "?" + q.Encode()); err != nil {
		return tokenResponse{}, fmt.Errorf("auth: open browser: %w", err)
	}

	var cb callback
	select {
	case cb = <-result:
	case <-ctx.Done():
		return tokenResponse{}, ctx.Err()
	}
	if cb.err != nil {
		return tokenResponse{}, cb.err
	}
	var r tokenResponse
	err = postForm(ctx, b.cfg.client(), b.d.ep.token, url.Values{
		"client_id":     {b.cfg.ClientID},
		"grant_type":    {"authorization_code"},
		"code":          {cb.code},
		"redirect_uri":  {redirect},
		"code_verifier": {verifier},
		"scope":         {scope},
	}, &r)
	return r, err
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}
