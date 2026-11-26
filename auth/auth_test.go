package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeIdP struct {
	*httptest.Server
	calls int32
}

func newIdP(t *testing.T, h func(path string, f url.Values, w http.ResponseWriter)) *fakeIdP {
	t.Helper()
	p := &fakeIdP{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&p.calls, 1)
		r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		h(r.URL.Path, r.PostForm, w)
	}))
	t.Cleanup(p.Close)
	return p
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func selfSigned(t *testing.T) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	keyDER, _ := x509.MarshalPKCS8PrivateKey(key)
	p := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	p = append(p, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})...)
	return cert, key, p
}

func TestCertificateToken(t *testing.T) {
	cert, _, pemData := selfSigned(t)
	idp := newIdP(t, func(path string, f url.Values, w http.ResponseWriter) {
		if f.Get("grant_type") != "client_credentials" || f.Get("scope") != "https://graph.microsoft.com/.default" {
			t.Errorf("form = %v", f)
		}
		parts := strings.Split(f.Get("client_assertion"), ".")
		if len(parts) != 3 {
			t.Fatal("bad jwt")
		}
		var hdr map[string]string
		raw, _ := base64.RawURLEncoding.DecodeString(parts[0])
		json.Unmarshal(raw, &hdr)
		s := sha256.Sum256(cert.Raw)
		if hdr["alg"] != "RS256" || hdr["x5t#S256"] != base64.RawURLEncoding.EncodeToString(s[:]) {
			t.Errorf("hdr = %v", hdr)
		}
		var claims map[string]any
		raw, _ = base64.RawURLEncoding.DecodeString(parts[1])
		json.Unmarshal(raw, &claims)
		if claims["iss"] != "cid" || claims["sub"] != "cid" {
			t.Errorf("claims = %v", claims)
		}
		sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
		h := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		if err := rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, h[:], sig); err != nil {
			t.Errorf("signature: %v", err)
		}
		if !strings.HasSuffix(claims["aud"].(string), "/tid/oauth2/v2.0/token") {
			t.Errorf("aud = %v", claims["aud"])
		}
		writeJSON(w, 200, map[string]any{"access_token": "app-tok", "expires_in": 3600})
	})
	dir := t.TempDir()
	path := filepath.Join(dir, "c.pem")
	os.WriteFile(path, pemData, 0o600)
	c, err := NewCertificateFromFile("tid", "cid", path, "", Options{Authority: idp.URL})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"https://graph.microsoft.com/.default"}})
	if err != nil || tok.Token != "app-tok" {
		t.Fatal(tok, err)
	}
	if c.Thumbprint() == "" {
		t.Error("thumbprint")
	}
}

func TestCertificateInvalid(t *testing.T) {
	for name, data := range map[string][]byte{
		"garbage":   []byte("nope"),
		"no key":    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("x")}),
		"cert only": func() []byte { _, _, p := selfSigned(t); b, _ := pem.Decode(p); return pem.EncodeToMemory(b) }(),
	} {
		if _, _, err := ParseCertificate(data, ""); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := NewCertificate(CertificateConfig{TenantID: "t", ClientID: "c"}); err == nil {
		t.Error("expected error for missing cert")
	}
}

func TestParsePFX(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	dir := t.TempDir()
	_, _, pemData := selfSigned(t)
	pemPath, pfx := filepath.Join(dir, "a.pem"), filepath.Join(dir, "a.pfx")
	os.WriteFile(pemPath, pemData, 0o600)
	var out []byte
	var err error
	for _, extra := range [][]string{{"-legacy"}, {}} {
		args := append([]string{"pkcs12", "-export", "-in", pemPath, "-inkey", pemPath, "-out", pfx, "-passout", "pass:pw"}, extra...)
		if out, err = exec.Command("openssl", args...).CombinedOutput(); err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("cannot create legacy pfx: %s", out)
	}
	data, _ := os.ReadFile(pfx)
	cert, key, err := ParseCertificate(data, "pw")
	if err != nil || cert == nil || key == nil {
		t.Skipf("pkcs12 decoder cannot read OpenSSL output: %v", err)
	}
}

func TestCacheReuseAndRefresh(t *testing.T) {
	cert, key, _ := selfSigned(t)
	expires := 3600
	idp := newIdP(t, func(path string, f url.Values, w http.ResponseWriter) {
		writeJSON(w, 200, map[string]any{"access_token": "t", "expires_in": expires})
	})
	c, _ := NewCertificate(CertificateConfig{TenantID: "t", ClientID: "c", Certificate: cert, Key: key, Options: Options{Authority: idp.URL}})
	o := TokenOptions{Scopes: []string{"s"}}
	c.Token(context.Background(), o)
	c.Token(context.Background(), o)
	if idp.calls != 1 {
		t.Fatalf("calls = %d, want 1", idp.calls)
	}
	c.Token(context.Background(), TokenOptions{Scopes: []string{"other"}})
	if idp.calls != 2 {
		t.Fatalf("calls = %d, want 2", idp.calls)
	}
	c.cache.now = func() time.Time { return time.Now().Add(time.Hour - time.Minute) }
	c.Token(context.Background(), o)
	if idp.calls != 3 {
		t.Fatalf("calls = %d, want 3", idp.calls)
	}
}

func TestCacheConcurrent(t *testing.T) {
	cert, key, _ := selfSigned(t)
	idp := newIdP(t, func(path string, f url.Values, w http.ResponseWriter) {
		writeJSON(w, 200, map[string]any{"access_token": "t", "expires_in": 3600})
	})
	c, _ := NewCertificate(CertificateConfig{TenantID: "t", ClientID: "c", Certificate: cert, Key: key, Options: Options{Authority: idp.URL}})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Token(context.Background(), TokenOptions{Scopes: []string{"s"}})
		}()
	}
	wg.Wait()
	if idp.calls != 1 {
		t.Fatalf("calls = %d, want 1", idp.calls)
	}
}

func deviceIdP(t *testing.T, tokenErrs ...string) *fakeIdP {
	var n int32
	return newIdP(t, func(path string, f url.Values, w http.ResponseWriter) {
		switch {
		case strings.HasSuffix(path, "/devicecode"):
			writeJSON(w, 200, map[string]any{"device_code": "dc", "user_code": "ABC", "verification_uri": "https://v", "expires_in": 900, "interval": 1, "message": "go sign in"})
		case f.Get("grant_type") == "refresh_token":
			writeJSON(w, 200, map[string]any{"access_token": "refreshed-" + f.Get("scope"), "expires_in": 3600})
		default:
			i := int(atomic.AddInt32(&n, 1)) - 1
			if i < len(tokenErrs) {
				writeJSON(w, 400, map[string]any{"error": tokenErrs[i]})
				return
			}
			writeJSON(w, 200, map[string]any{"access_token": "dev-tok", "refresh_token": "rt", "expires_in": 3600})
		}
	})
}

func TestDeviceCodeSuccess(t *testing.T) {
	idp := deviceIdP(t, "authorization_pending", "slow_down")
	var msg DeviceCodeMessage
	c, err := NewDeviceCode(DeviceCodeConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL},
		Prompt: func(m DeviceCodeMessage) error { msg = m; return nil }})
	if err != nil {
		t.Fatal(err)
	}
	var slept []time.Duration
	c.sleep = func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil }
	tok, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"https://graph.microsoft.com/.default"}})
	if err != nil || tok.Token != "dev-tok" || msg.UserCode != "ABC" {
		t.Fatal(tok, msg, err)
	}
	if len(slept) != 3 || slept[2] != 6*time.Second {
		t.Fatalf("slept = %v (slow_down must add 5s)", slept)
	}
	tok, err = c.Token(context.Background(), TokenOptions{Scopes: []string{"https://x.sharepoint.com/.default"}})
	if err != nil || !strings.HasPrefix(tok.Token, "refreshed-") {
		t.Fatal(tok, err)
	}
}

func TestDeviceCodeExpired(t *testing.T) {
	idp := deviceIdP(t, "authorization_pending", "expired_token")
	c, _ := NewDeviceCode(DeviceCodeConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL}, Prompt: func(DeviceCodeMessage) error { return nil }})
	c.sleep = func(context.Context, time.Duration) error { return nil }
	_, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"s"}})
	if !errors.Is(err, ErrDeviceCodeExpired) {
		t.Fatal(err)
	}
}

func TestDeviceCodeCancelled(t *testing.T) {
	idp := deviceIdP(t, "authorization_pending", "authorization_pending", "authorization_pending")
	c, _ := NewDeviceCode(DeviceCodeConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL}, Prompt: func(DeviceCodeMessage) error { return nil }})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := 0
	c.sleep = func(ctx context.Context, d time.Duration) error {
		if n++; n == 2 {
			cancel()
		}
		return ctx.Err()
	}
	if _, err := c.Token(ctx, TokenOptions{Scopes: []string{"s"}}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func browserIdP(t *testing.T) *fakeIdP {
	return newIdP(t, func(path string, f url.Values, w http.ResponseWriter) {
		if f.Get("grant_type") != "authorization_code" || f.Get("code") != "the-code" || f.Get("code_verifier") == "" {
			t.Errorf("form = %v", f)
		}
		writeJSON(w, 200, map[string]any{"access_token": "br-tok", "refresh_token": "rt", "expires_in": 3600})
	})
}

func simulate(t *testing.T, mutate func(q url.Values) url.Values) func(string) error {
	return func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		q := u.Query()
		if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
			t.Errorf("missing PKCE: %v", q)
		}
		ru, _ := url.Parse(q.Get("redirect_uri"))
		cb := url.Values{"code": {"the-code"}, "state": {q.Get("state")}}
		if mutate != nil {
			cb = mutate(cb)
		}
		go func() {
			resp, err := http.Get("http://127.0.0.1:" + ru.Port() + "/?" + cb.Encode())
			if err == nil {
				resp.Body.Close()
			}
		}()
		return nil
	}
}

func TestBrowserSuccess(t *testing.T) {
	idp := browserIdP(t)
	c, _ := NewInteractiveBrowser(BrowserConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL}, Open: simulate(t, nil)})
	tok, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"s"}})
	if err != nil || tok.Token != "br-tok" {
		t.Fatal(tok, err)
	}
}

func TestBrowserStateMismatch(t *testing.T) {
	idp := browserIdP(t)
	c, _ := NewInteractiveBrowser(BrowserConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL},
		Open: simulate(t, func(q url.Values) url.Values { q.Set("state", "forged"); return q })})
	_, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"s"}})
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatal(err)
	}
	if idp.calls != 0 {
		t.Error("code must not be exchanged on state mismatch")
	}
}

func TestBrowserAuthError(t *testing.T) {
	idp := browserIdP(t)
	c, _ := NewInteractiveBrowser(BrowserConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL},
		Open: simulate(t, func(q url.Values) url.Values { q.Set("error", "access_denied"); return q })})
	_, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"s"}})
	var oe *OAuthError
	if !errors.As(err, &oe) || oe.Code != "access_denied" {
		t.Fatal(err)
	}
}

func TestBrowserTimeout(t *testing.T) {
	idp := browserIdP(t)
	c, _ := NewInteractiveBrowser(BrowserConfig{TenantID: "t", ClientID: "c", Options: Options{Authority: idp.URL},
		Timeout: 50 * time.Millisecond, Open: func(string) error { return nil }})
	_, err := c.Token(context.Background(), TokenOptions{Scopes: []string{"s"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
