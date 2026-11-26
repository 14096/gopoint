package sharepoint_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/sharepoint"
	"github.com/14096/gopoint/sharepoint/spx"
)

type fakeCred struct{ scopes []string }

func (f *fakeCred) Token(_ context.Context, o auth.TokenOptions) (auth.AccessToken, error) {
	f.scopes = o.Scopes
	return auth.AccessToken{Token: "tok"}, nil
}

func TestClientDigestAndScope(t *testing.T) {
	var digests int32
	var writes []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("Accept") != "application/json;odata=nometadata" {
			t.Errorf("headers = %v", r.Header)
		}
		switch {
		case r.URL.Path == "/sites/t/_api/contextinfo":
			atomic.AddInt32(&digests, 1)
			w.Write([]byte(`{"FormDigestValue":"dig","FormDigestTimeoutSeconds":1800}`))
		case r.Method == "POST":
			writes = append(writes, r.Header.Get("X-RequestDigest"))
			w.Write([]byte(`{"Id":"x"}`))
		default:
			if r.Header.Get("X-RequestDigest") != "" {
				t.Error("digest sent on GET")
			}
			w.Write([]byte(`{"Title":"Team"}`))
		}
	}))
	defer srv.Close()

	cred := &fakeCred{}
	c, err := sharepoint.New(srv.URL+"/sites/t/", cred, sharepoint.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if w, err := c.Web().Get(ctx); err != nil || w.Title != "Team" {
		t.Fatal(w, err)
	}
	if want := "https://" + srv.Listener.Addr().String() + "/.default"; len(cred.scopes) != 1 || cred.scopes[0] != want {
		t.Fatalf("scopes = %v, want %s", cred.scopes, want)
	}
	for i := 0; i < 2; i++ {
		if err := c.Lists().Update(ctx, spx.ByTitle("L"), map[string]any{"Title": "x"}); err != nil {
			t.Fatal(err)
		}
	}
	if digests != 1 || len(writes) != 2 || writes[0] != "dig" {
		t.Fatalf("digests=%d writes=%v", digests, writes)
	}
	var out map[string]any
	if err := c.Raw().Get(ctx, "web/currentuser", &out); err != nil || out["Title"] != "Team" {
		t.Fatal(out, err)
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := sharepoint.New("not a url", &fakeCred{}, sharepoint.Options{}); err == nil {
		t.Error("want error for bad url")
	}
	if _, err := sharepoint.New("https://x.sharepoint.com", nil, sharepoint.Options{}); err == nil {
		t.Error("want error for nil credential")
	}
}
