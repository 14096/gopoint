package permissions_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/permissions"
	"github.com/14096/gopoint/internal/testutil"
)

func TestPermissions(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.Write([]byte(`{"value":[{"id":"p","roles":["read"],"link":{"type":"view","webUrl":"https://l"}}],"id":"p","roles":["write"]}`))
	})
	s := permissions.New(srv.Client("/v1.0"))
	ctx := context.Background()

	ps, err := s.All(ctx, "d", "i")
	if err != nil || ps[0].Link.WebURL != "https://l" || srv.Last().Path != "/v1.0/drives/d/items/i/permissions" {
		t.Fatal(ps, err)
	}
	n := 0
	for _, err := range s.Iter(ctx, "d", "i") {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
	if p, err := s.Get(ctx, "d", "i", "p"); err != nil || p.ID != "p" {
		t.Fatal(p, err)
	}
	if p, err := s.Update(ctx, "d", "i", "p", permissions.Write); err != nil || srv.Last().Method != "PATCH" || srv.Last().Body != `{"roles":["write"]}` {
		t.Fatal(p, err)
	}
	if err := s.Delete(ctx, "d", "i", "p"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AllSite(ctx, "s"); err != nil || srv.Last().Path != "/v1.0/sites/s/permissions" {
		t.Fatal(err)
	}
	if _, err := s.GrantSiteApp(ctx, "s", "app", "App", permissions.Write); err != nil ||
		srv.Last().Body != `{"grantedToIdentities":[{"application":{"displayName":"App","id":"app"}}],"roles":["write"]}` {
		t.Fatal(err, srv.Last().Body)
	}
	if err := s.DeleteSite(ctx, "s", "p"); err != nil || srv.Last().Path != "/v1.0/sites/s/permissions/p" {
		t.Fatal(err)
	}
}
