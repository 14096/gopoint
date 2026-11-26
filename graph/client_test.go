package graph_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/graph"
)

type fakeCred struct{ scopes []string }

func (f *fakeCred) Token(_ context.Context, o auth.TokenOptions) (auth.AccessToken, error) {
	f.scopes = o.Scopes
	return auth.AccessToken{Token: "tok"}, nil
}

func TestClientScopeBaseURLAndRaw(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1.0/sites/root/permissions" || r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("%s %v", r.URL.Path, r.Header)
		}
		w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()
	cred := &fakeCred{}
	c, err := graph.New(cred, graph.Options{BaseURL: srv.URL + "/v1.0"})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := c.Raw().Get(context.Background(), "/sites/root/permissions", &out); err != nil {
		t.Fatal(err)
	}
	if want := "https://" + srv.Listener.Addr().String() + "/.default"; cred.scopes[0] != want {
		t.Fatalf("scope = %v", cred.scopes)
	}
	if c.Sites() == nil || c.Lists() == nil || c.Columns() == nil || c.Items() == nil || c.ContentTypes() == nil || c.Drives() == nil {
		t.Fatal("missing resource")
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := graph.New(nil, graph.Options{}); err == nil {
		t.Error("want error for nil credential")
	}
	if _, err := graph.New(&fakeCred{}, graph.Options{BaseURL: "::"}); err == nil {
		t.Error("want error for bad URL")
	}
	c, err := graph.New(&fakeCred{}, graph.Options{})
	if err != nil || c.Raw().BaseURL() != graph.DefaultBaseURL {
		t.Fatal(err)
	}
}
