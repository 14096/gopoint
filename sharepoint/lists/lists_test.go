package lists_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/lists"
	"github.com/14096/gopoint/sharepoint/spx"
)

func TestListCRUD(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case c.Path == "/_api/web/lists/getbytitle('Missing')":
			w.WriteHeader(404)
			w.Write([]byte(`{"odata.error":{"code":"-2130575322, Microsoft.SharePoint.SPException","message":{"lang":"en-US","value":"List does not exist."}}}`))
		case c.Method == "POST" && c.Path == "/_api/web/lists":
			w.Write([]byte(`{"Id":"guid-1","Title":"Tasks","BaseTemplate":100}`))
		default:
			w.Write([]byte(`{"Id":"guid-1","Title":"My List","ItemCount":3}`))
		}
	})
	s := lists.New(srv.Client("/_api"))
	ctx := context.Background()

	l, err := s.Get(ctx, spx.ByTitle("My List"))
	if err != nil || l.ItemCount != 3 || srv.Last().Path != "/_api/web/lists/getbytitle('My List')" {
		t.Fatal(l, err, srv.Last().Path)
	}
	if _, err = s.Get(ctx, spx.ByID("abc")); err != nil || srv.Last().Path != "/_api/web/lists(guid'abc')" {
		t.Fatal(err, srv.Last().Path)
	}
	c, err := s.Create(ctx, lists.CreateParams{Title: "Tasks"})
	if err != nil || c.ID != "guid-1" {
		t.Fatal(c, err)
	}
	if b := srv.Last().Body; !contains(b, `"BaseTemplate":100`) {
		t.Fatal(b)
	}
	if err := s.Update(ctx, spx.ByTitle("My List"), map[string]any{"Title": "N"}); err != nil {
		t.Fatal(err)
	}
	if h := srv.Last().Header; h.Get("X-HTTP-Method") != "MERGE" || h.Get("If-Match") != "*" {
		t.Fatal(h)
	}
	if err := s.Delete(ctx, spx.ByTitle("My List")); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
	_, err = s.Get(ctx, spx.ByTitle("Missing"))
	he, ok := err.(*httpx.Error)
	if !ok || he.Status != 404 || he.Message != "List does not exist." {
		t.Fatalf("%#v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
