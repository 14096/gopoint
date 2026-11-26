package views_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/spx"
	"github.com/14096/gopoint/sharepoint/views"
)

func TestViews(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if strings.HasSuffix(c.Path, "/viewfields") {
			w.Write([]byte(`{"Items":["Title","Status"]}`))
			return
		}
		w.Write([]byte(`{"value":[{"Id":"v1","Title":"All"}],"Id":"v1","Title":"All","RowLimit":30}`))
	})
	s := views.New(srv.Client("/_api"))
	ctx := context.Background()
	l := spx.ByTitle("Tasks")
	const v = "/_api/web/lists/getbytitle('Tasks')/views(guid'v1')"

	if vs, err := s.All(ctx, l); err != nil || len(vs) != 1 || srv.Last().Path != "/_api/web/lists/getbytitle('Tasks')/views" {
		t.Fatal(vs, err)
	}
	if g, err := s.Get(ctx, l, "v1"); err != nil || g.RowLimit != 30 || srv.Last().Path != v {
		t.Fatal(g, err)
	}
	if _, err := s.ByTitle(ctx, l, "All"); err != nil || !strings.HasSuffix(srv.Last().Path, "/views/getbytitle('All')") {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, l, views.CreateParams{Title: "Open", Fields: []string{"Title"}, RowLimit: 10}); err != nil ||
		srv.Last().Body != `{"Paged":false,"PersonalView":false,"RowLimit":10,"SetAsDefaultView":false,"Title":"Open","ViewFields":["Title"]}` {
		t.Fatal(err, srv.Last().Body)
	}
	title, n := "New", 5
	if err := s.Update(ctx, l, "v1", views.UpdateParams{Title: &title, RowLimit: &n}); err != nil ||
		srv.Last().Header.Get("X-HTTP-Method") != "MERGE" || srv.Last().Body != `{"RowLimit":5,"Title":"New"}` {
		t.Fatal(err, srv.Last().Body)
	}
	if fs, err := s.Fields(ctx, l, "v1"); err != nil || len(fs) != 2 {
		t.Fatal(fs, err)
	}
	if err := s.MoveField(ctx, l, "v1", "Status", 0); err != nil || srv.Last().Body != `{"field":"Status","index":0}` {
		t.Fatal(err)
	}
	if err := s.SetFields(ctx, l, "v1", []string{"A", "B"}); err != nil || srv.Last().Path != v+"/viewfields/addviewfield('B')" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.RemoveField(ctx, l, "v1", "A"); err != nil || !strings.HasSuffix(srv.Last().Path, "/removeviewfield('A')") {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, l, "v1"); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
}
