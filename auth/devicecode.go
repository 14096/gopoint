package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"
)

// ErrDeviceCodeExpired is returned when the user did not sign in in time.
var ErrDeviceCodeExpired = errors.New("auth: device code expired before sign-in completed")

// DeviceCodeMessage is what the user must be shown.
type DeviceCodeMessage struct {
	UserCode        string
	VerificationURI string
	Message         string
	ExpiresIn       time.Duration
}

// DeviceCodeConfig configures a device code credential.
type DeviceCodeConfig struct {
	TenantID string // tenant ID/domain, or "organizations"
	ClientID string
	// Prompt shows the sign-in instructions. Defaults to printing to stderr.
	Prompt func(DeviceCodeMessage) error
	Options
}

// DeviceCode is a delegated credential using the OAuth2 device code flow.
type DeviceCode struct {
	d     *delegated
	cfg   DeviceCodeConfig
	sleep func(context.Context, time.Duration) error
}

// NewDeviceCode returns a device code credential.
func NewDeviceCode(cfg DeviceCodeConfig) (*DeviceCode, error) {
	if cfg.TenantID == "" || cfg.ClientID == "" {
		return nil, errors.New("auth: TenantID and ClientID are required")
	}
	if cfg.Prompt == nil {
		cfg.Prompt = func(m DeviceCodeMessage) error {
			_, err := fmt.Fprintln(os.Stderr, m.Message)
			return err
		}
	}
	dc := &DeviceCode{cfg: cfg, sleep: sleepCtx}
	dc.d = &delegated{
		opts: cfg.Options, ep: newEndpoints(cfg.authority(), cfg.TenantID),
		clientID: cfg.ClientID, interactive: dc.signIn,
	}
	return dc, nil
}

// Token implements Credential.
func (c *DeviceCode) Token(ctx context.Context, o TokenOptions) (AccessToken, error) {
	return c.d.Token(ctx, o)
}

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

func (c *DeviceCode) signIn(ctx context.Context, scope string) (tokenResponse, error) {
	var dc struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Message         string `json:"message"`
	}
	hc := c.cfg.client()
	if err := postForm(ctx, hc, c.d.ep.deviceCode, url.Values{
		"client_id": {c.cfg.ClientID}, "scope": {scope},
	}, &dc); err != nil {
		return tokenResponse{}, err
	}
	expires := time.Duration(dc.ExpiresIn) * time.Second
	if err := c.cfg.Prompt(DeviceCodeMessage{
		UserCode: dc.UserCode, VerificationURI: dc.VerificationURI, Message: dc.Message, ExpiresIn: expires,
	}); err != nil {
		return tokenResponse{}, err
	}
	interval := time.Duration(dc.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(expires)
	for {
		if err := c.sleep(ctx, interval); err != nil {
			return tokenResponse{}, err
		}
		if expires > 0 && time.Now().After(deadline) {
			return tokenResponse{}, ErrDeviceCodeExpired
		}
		var r tokenResponse
		err := postForm(ctx, hc, c.d.ep.token, url.Values{
			"client_id":   {c.cfg.ClientID},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {dc.DeviceCode},
		}, &r)
		if err == nil {
			return r, nil
		}
		var oe *OAuthError
		if !errors.As(err, &oe) {
			return tokenResponse{}, err
		}
		switch oe.Code {
		case "authorization_pending":
		case "slow_down":
			interval += 5 * time.Second
		case "expired_token", "code_expired":
			return tokenResponse{}, ErrDeviceCodeExpired
		default:
			return tokenResponse{}, err
		}
	}
}
