package pages_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/pages"
	"github.com/14096/gopoint/internal/testutil"
)

func TestPages(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"value":[{"id":"p1","title":"Home"}],"id":"p1","title":"Home"}`))
	})
	s := pages.New(srv.Client("/v1.0"))
	ctx := context.Background()
	n := 0
	for p, err := range s.Iter(ctx, "s1") {
		if err != nil || p.ID != "p1" {
			t.Fatal(p, err)
		}
		n++
	}
	if n != 1 || srv.Last().Path != "/v1.0/sites/s1/pages/microsoft.graph.sitePage" {
		t.Fatal(n, srv.Last().Path)
	}
	if ps, err := s.All(ctx, "s1"); err != nil || len(ps) != 1 {
		t.Fatal(ps, err)
	}
	if p, err := s.Get(ctx, "s1", "p1"); err != nil || p.Title != "Home" || srv.Last().Path != "/v1.0/sites/s1/pages/p1/microsoft.graph.sitePage" {
		t.Fatal(p, err)
	}
	if _, err := s.Create(ctx, "s1", pages.CreateParams{Name: "a.aspx", Title: "A"}); err != nil || srv.Last().Method != "POST" ||
		srv.Last().Body != `{"@odata.type":"#microsoft.graph.sitePage","name":"a.aspx","pageLayout":"article","title":"A"}` {
		t.Fatal(err, srv.Last().Body)
	}
	if _, err := s.Update(ctx, "s1", "p1", "B"); err != nil || srv.Last().Method != "PATCH" {
		t.Fatal(err)
	}
	if err := s.Publish(ctx, "s1", "p1"); err != nil || srv.Last().Path != "/v1.0/sites/s1/pages/p1/microsoft.graph.sitePage/publish" {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "s1", "p1"); err != nil || srv.Last().Method != "DELETE" {
		t.Fatal(err)
	}
}
