package pages_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/pages"
)

func TestPages(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"value":[{"Id":3,"Title":"Home"}],"Id":3,"Title":"Home"}`))
	})
	s := pages.New(srv.Client("/_api"))
	ctx := context.Background()

	if ps, err := s.All(ctx, httpx.ODataQuery{}); err != nil || len(ps) != 1 {
		t.Fatal(ps, err)
	}
	n := 0
	for _, err := range s.Iter(ctx, httpx.ODataQuery{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
	if p, err := s.Get(ctx, 3); err != nil || p.Title != "Home" || srv.Last().Path != "/_api/sitepages/pages(3)" {
		t.Fatal(p, err, srv.Last().Path)
	}
	if _, err := s.GetByURL(ctx, "SitePages/Home.aspx"); err != nil || srv.Last().Path != "/_api/sitepages/pages/GetByUrl('SitePages/Home.aspx')" {
		t.Fatal(err, srv.Last().Path)
	}
	if _, err := s.Create(ctx, pages.CreateParams{Title: "T", Name: "t.aspx"}); err != nil ||
		srv.Last().Body != `{"Name":"t.aspx","PageLayoutType":"Article","Title":"T"}` {
		t.Fatal(err, srv.Last().Body)
	}
	if _, err := s.Update(ctx, 3, pages.UpdateParams{Title: "X"}); err != nil || srv.Last().Path != "/_api/sitepages/pages(3)/savepagedraft" {
		t.Fatal(err)
	}
	if err := s.Publish(ctx, 3); err != nil || srv.Last().Path != "/_api/sitepages/pages(3)/publish" {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, 3); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
}
