package hubsites_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/hubsites"
)

func TestHubSites(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Path == "/_api/site" {
			w.Write([]byte(`{"HubSiteId":"abc"}`))
			return
		}
		w.Write([]byte(`{"value":[{"ID":"h1","Title":"Hub"}],"ID":"h1","Title":"Hub"}`))
	})
	s := hubsites.New(srv.Client("/_api"))
	ctx := context.Background()

	if hs, err := s.All(ctx); err != nil || hs[0].ID != "h1" || srv.Last().Path != "/_api/hubsites" {
		t.Fatal(hs, err)
	}
	if h, err := s.Get(ctx, "h1"); err != nil || h.Title != "Hub" || srv.Last().Path != "/_api/hubsites(guid'h1')" {
		t.Fatal(h, err)
	}
	if id, err := s.Current(ctx); err != nil || id != "abc" {
		t.Fatal(id, err)
	}
	if h, err := s.Register(ctx); err != nil || h.ID != "h1" || srv.Last().Path != "/_api/site/registerhubsite" {
		t.Fatal(h, err)
	}
	if err := s.Unregister(ctx); err != nil || srv.Last().Path != "/_api/site/unregisterhubsite" {
		t.Fatal(err)
	}
	title := "New"
	if err := s.Update(ctx, "h1", hubsites.UpdateParams{Title: &title}); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "MERGE" || srv.Last().Body != `{"Title":"New"}` {
		t.Fatal(err)
	}
	if err := s.Join(ctx, "h1"); err != nil || srv.Last().Path != "/_api/site/joinhubsite('h1')" {
		t.Fatal(err)
	}
	if err := s.Leave(ctx); err != nil || srv.Last().Path != "/_api/site/joinhubsite('"+hubsites.NoHub+"')" {
		t.Fatal(err)
	}
}
