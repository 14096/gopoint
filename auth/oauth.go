package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// OAuthError is an error response from the identity platform.
type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *OAuthError) Error() string {
	if e.Description == "" {
		return "auth: " + e.Code
	}
	return fmt.Sprintf("auth: %s: %s", e.Code, e.Description)
}

type tokenResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    json.Number `json:"expires_in"`
	OAuthError
}

func (r tokenResponse) accessToken(now time.Time) AccessToken {
	n, _ := r.ExpiresIn.Int64()
	return AccessToken{Token: r.AccessToken, ExpiresOn: now.Add(time.Duration(n) * time.Second)}
}

type endpoints struct{ token, deviceCode, authorize string }

func newEndpoints(authority, tenant string) endpoints {
	base := authority + "/" + url.PathEscape(tenant) + "/oauth2/v2.0/"
	return endpoints{token: base + "token", deviceCode: base + "devicecode", authorize: base + "authorize"}
}

// postForm posts a form and decodes either a success or an OAuthError.
func postForm(ctx context.Context, hc *http.Client, endpoint string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		var oe OAuthError
		if json.Unmarshal(b, &oe) != nil || oe.Code == "" {
			return fmt.Errorf("auth: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		}
		return &oe
	}
	return json.Unmarshal(b, out)
}

// delegated implements the shared token/refresh logic of user flows. The
// interactive function runs the flow-specific sign-in for a scope string.
type delegated struct {
	opts        Options
	ep          endpoints
	clientID    string
	interactive func(ctx context.Context, scope string) (tokenResponse, error)

	cache   tokenCache
	refresh string // guarded by cache.mu (only touched inside fetch)
}

func (d *delegated) Token(ctx context.Context, o TokenOptions) (AccessToken, error) {
	if len(o.Scopes) == 0 {
		return AccessToken{}, errors.New("auth: at least one scope is required")
	}
	scope := strings.Join(slices.Concat(o.Scopes, []string{"offline_access"}), " ")
	return d.cache.get(scopeKey(o.Scopes), func() (AccessToken, error) {
		if d.refresh != "" {
			var r tokenResponse
			err := postForm(ctx, d.opts.client(), d.ep.token, url.Values{
				"client_id":     {d.clientID},
				"grant_type":    {"refresh_token"},
				"refresh_token": {d.refresh},
				"scope":         {scope},
			}, &r)
			if err == nil {
				if r.RefreshToken != "" {
					d.refresh = r.RefreshToken
				}
				return r.accessToken(time.Now()), nil
			}
			var oe *OAuthError
			if !errors.As(err, &oe) || oe.Code != "invalid_grant" {
				return AccessToken{}, err
			}
			d.refresh = ""
		}
		r, err := d.interactive(ctx, scope)
		if err != nil {
			return AccessToken{}, err
		}
		if r.RefreshToken != "" {
			d.refresh = r.RefreshToken
		}
		return r.accessToken(time.Now()), nil
	})
}
