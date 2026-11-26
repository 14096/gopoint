package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTest(t *testing.T, h http.HandlerFunc, cfg Config) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cfg.BaseURL = srv.URL + "/api"
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c, srv
}

func TestRequestComposition(t *testing.T) {
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/things" || r.URL.Query().Get("$top") != "3" {
			t.Errorf("url = %s", r.URL)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Error("missing auth")
		}
		if r.Header.Get("Accept") != "text/x" || r.Header.Get("X-Def") != "d" {
			t.Errorf("headers = %v", r.Header)
		}
		w.Write([]byte(`{"a":1}`))
	}, Config{
		Token:  func(context.Context) (string, error) { return "tok", nil },
		Header: http.Header{"X-Def": {"d"}, "Accept": {"application/json"}},
	})
	req := NewRequest("GET", "things").SetQuery("$top", "3").SetHeader("Accept", "text/x")
	resp, err := c.Do(context.Background(), req)
	if err != nil || string(resp.Body) != `{"a":1}` {
		t.Fatal(resp, err)
	}
}

func TestGenericCall(t *testing.T) {
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || string(b) != `{"x":1}` {
			t.Errorf("%s %s", r.Method, b)
		}
		w.Write([]byte(`{"ok":true}`))
	}, Config{})
	var out struct{ OK bool }
	if err := c.Post(context.Background(), "/p", map[string]int{"x": 1}, &out); err != nil || !out.OK {
		t.Fatal(out, err)
	}
}

func TestForeignHostRefused(t *testing.T) {
	c, err := New(Config{BaseURL: "https://a.example/v1", Token: func(context.Context) (string, error) { return "t", nil }})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Do(context.Background(), NewRequest("GET", "https://evil.example/x"))
	if err == nil || !strings.Contains(err.Error(), "foreign host") {
		t.Fatal(err)
	}
}

func TestNewInvalidBaseURL(t *testing.T) {
	if _, err := New(Config{BaseURL: "http://[::1"}); err == nil {
		t.Fatal("expected error for invalid BaseURL")
	}
}

func TestRetryThrottled(t *testing.T) {
	var n int32
	var bodies []string
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(429)
			return
		}
		w.Write([]byte(`ok`))
	}, Config{})
	var slept []time.Duration
	c.sleep = func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil }
	req := NewRequest("POST", "x").WithBytes([]byte("payload"), "")
	if _, err := c.Do(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if len(slept) != 1 || slept[0] < 2*time.Second {
		t.Fatalf("slept = %v", slept)
	}
	if len(bodies) != 2 || bodies[0] != "payload" || bodies[1] != "payload" {
		t.Fatalf("bodies = %q", bodies)
	}
}

func TestRetryExhausted(t *testing.T) {
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }, Config{MaxAttempts: 3})
	c.sleep = func(context.Context, time.Duration) error { return nil }
	_, err := c.Do(context.Background(), NewRequest("GET", "x"))
	var he *Error
	if !errors.As(err, &he) || he.Status != 503 {
		t.Fatal(err)
	}
}

func TestSleepCancelled(t *testing.T) {
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) }, Config{})
	ctx, cancel := context.WithCancel(context.Background())
	c.sleep = func(ctx context.Context, d time.Duration) error { cancel(); return sleepCtx(ctx, time.Hour) }
	if _, err := c.Do(ctx, NewRequest("GET", "x")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestErrorFormats(t *testing.T) {
	cases := map[string]struct {
		body, code, msg string
	}{
		"graph":   {`{"error":{"code":"itemNotFound","message":"gone"}}`, "itemNotFound", "gone"},
		"verbose": {`{"error":{"code":"-1, Foo","message":{"lang":"en","value":"bad"}}}`, "-1, Foo", "bad"},
		"nometa":  {`{"odata.error":{"code":"-2, Bar","message":{"lang":"en","value":"worse"}}}`, "-2, Bar", "worse"},
		"nonjson": {`oops`, "", "oops"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("request-id", "rid")
				w.WriteHeader(404)
				w.Write([]byte(tc.body))
			}, Config{})
			_, err := c.Do(context.Background(), NewRequest("GET", "x"))
			he, ok := err.(*Error)
			if !ok || he.Status != 404 || he.Code != tc.code || he.Message != tc.msg || he.RequestID != "rid" {
				t.Fatalf("%#v", err)
			}
			if !IsNotFound(err) {
				t.Fatal("IsNotFound")
			}
		})
	}
}
