package sites_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/sites"
	"github.com/14096/gopoint/internal/testutil"
)

func TestSites(t *testing.T) {
	var srv *testutil.Server
	srv = testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case c.Path == "/v1.0/sites/contoso.sharepoint.com:/sites/team":
			w.Write([]byte(`{"id":"contoso.sharepoint.com,g1,g2","webUrl":"https://contoso.sharepoint.com/sites/team"}`))
		case c.Path == "/v1.0/sites" && c.Query.Get("page") == "":
			fmt.Fprintf(w, `{"value":[{"id":"1"}],"@odata.nextLink":"%s/v1.0/sites?search=x&page=2"}`, srv.URL)
		case c.Path == "/v1.0/sites":
			w.Write([]byte(`{"value":[{"id":"2"}]}`))
		default:
			w.Write([]byte(`{"id":"root","displayName":"Root"}`))
		}
	})
	s := sites.New(srv.Client("/v1.0"))
	ctx := context.Background()
	site, err := s.ByPath(ctx, "contoso.sharepoint.com", "sites/team")
	if err != nil || site.ID != "contoso.sharepoint.com,g1,g2" || site.WebURL == "" {
		t.Fatal(site, err)
	}
	if r, err := s.Root(ctx); err != nil || r.DisplayName != "Root" {
		t.Fatal(r, err)
	}
	if _, err := s.Get(ctx, "contoso.sharepoint.com,g1,g2"); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for it, err := range s.Search(ctx, "x") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, it.ID)
	}
	if len(ids) != 2 || srv.Calls[len(srv.Calls)-2].Query.Get("search") != "x" {
		t.Fatal(ids)
	}
	if _, err := s.Subsites(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
}
