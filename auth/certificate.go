package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/pkcs12"
)

// CertificateConfig configures an app-only certificate credential.
type CertificateConfig struct {
	TenantID string
	ClientID string
	// Certificate is the parsed certificate matching Key.
	Certificate *x509.Certificate
	Key         *rsa.PrivateKey
	Options
}

// Certificate authenticates as an application by signing a client assertion.
type Certificate struct {
	cfg   CertificateConfig
	ep    endpoints
	cache tokenCache
}

// NewCertificate validates cfg and returns the credential. No network call is made.
func NewCertificate(cfg CertificateConfig) (*Certificate, error) {
	switch {
	case cfg.TenantID == "" || cfg.ClientID == "":
		return nil, errors.New("auth: TenantID and ClientID are required")
	case cfg.Certificate == nil || cfg.Key == nil:
		return nil, errors.New("auth: certificate and private key are required")
	}
	return &Certificate{cfg: cfg, ep: newEndpoints(cfg.authority(), cfg.TenantID)}, nil
}

// NewCertificateFromFile loads a PEM or PFX file and builds the credential.
// password is only used for PFX files.
func NewCertificateFromFile(tenantID, clientID, path, password string, opts Options) (*Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cert, key, err := ParseCertificate(data, password)
	if err != nil {
		return nil, err
	}
	return NewCertificate(CertificateConfig{TenantID: tenantID, ClientID: clientID, Certificate: cert, Key: key, Options: opts})
}

// ParseCertificate parses PEM (certificate + private key) or PFX data.
func ParseCertificate(data []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if block, _ := pem.Decode(data); block != nil {
		return parsePEM(data)
	}
	key, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, nil, fmt.Errorf("auth: cannot parse certificate (neither PEM nor PFX): %w", err)
	}
	rk, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("auth: only RSA private keys are supported")
	}
	return cert, rk, nil
}

func parsePEM(data []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	var certs []*x509.Certificate
	var key *rsa.PrivateKey
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("auth: parse certificate: %w", err)
			}
			certs = append(certs, c)
		case "PRIVATE KEY":
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("auth: parse private key: %w", err)
			}
			rk, ok := k.(*rsa.PrivateKey)
			if !ok {
				return nil, nil, errors.New("auth: only RSA private keys are supported")
			}
			key = rk
		case "RSA PRIVATE KEY":
			k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, nil, fmt.Errorf("auth: parse private key: %w", err)
			}
			key = k
		case "ENCRYPTED PRIVATE KEY":
			return nil, nil, errors.New("auth: encrypted PEM keys are not supported; use a PFX file")
		}
	}
	if key == nil {
		return nil, nil, errors.New("auth: no private key found")
	}
	for _, c := range certs {
		if pub, ok := c.PublicKey.(*rsa.PublicKey); ok && pub.Equal(&key.PublicKey) {
			return c, key, nil
		}
	}
	return nil, nil, errors.New("auth: no certificate matching the private key found")
}

// Thumbprint returns the SHA-1 hex thumbprint shown in the Entra portal.
func (c *Certificate) Thumbprint() string {
	s := sha1.Sum(c.cfg.Certificate.Raw)
	return strings.ToUpper(hex.EncodeToString(s[:]))
}

func (c *Certificate) assertion(now time.Time) (string, error) {
	b64 := base64.RawURLEncoding.EncodeToString
	s256 := sha256.Sum256(c.cfg.Certificate.Raw)
	s1 := sha1.Sum(c.cfg.Certificate.Raw)
	header, _ := json.Marshal(map[string]string{
		"alg": "RS256", "typ": "JWT", "x5t#S256": b64(s256[:]), "x5t": b64(s1[:]),
	})
	claims, _ := json.Marshal(map[string]any{
		"aud": c.ep.token, "iss": c.cfg.ClientID, "sub": c.cfg.ClientID,
		"jti": rand.Text(),
		"nbf": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(),
	})
	signing := b64(header) + "." + b64(claims)
	h := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.cfg.Key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return signing + "." + b64(sig), nil
}

// Token implements Credential.
func (c *Certificate) Token(ctx context.Context, o TokenOptions) (AccessToken, error) {
	if len(o.Scopes) == 0 {
		return AccessToken{}, errors.New("auth: at least one scope is required")
	}
	return c.cache.get(scopeKey(o.Scopes), func() (AccessToken, error) {
		assertion, err := c.assertion(time.Now())
		if err != nil {
			return AccessToken{}, err
		}
		var r tokenResponse
		err = postForm(ctx, c.cfg.client(), c.ep.token, url.Values{
			"client_id":             {c.cfg.ClientID},
			"grant_type":            {"client_credentials"},
			"scope":                 {strings.Join(o.Scopes, " ")},
			"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
			"client_assertion":      {assertion},
		}, &r)
		if err != nil {
			return AccessToken{}, err
		}
		return r.accessToken(time.Now()), nil
	})
}
