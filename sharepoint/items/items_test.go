package items_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/items"
	"github.com/14096/gopoint/sharepoint/spx"
)

func TestFilteredQueryFollowsPages(t *testing.T) {
	var srv *testutil.Server
	srv = testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Query["page"] == nil {
			fmt.Fprintf(w, `{"value":[{"Id":1,"Title":"a"}],"odata.nextLink":"%s/_api/web/lists/getbytitle('L')/items?page=2"}`, srv.URL)
			return
		}
		fmt.Fprint(w, `{"value":[{"Id":2,"Title":"b"}]}`)
	})
	s := items.New(srv.Client("/_api"))
	got, err := s.All(context.Background(), spx.ByTitle("L"), httpx.ODataQuery{
		Select: []string{"Id", "Title"}, Filter: "Title ne 'x'", Top: 1, OrderBy: []string{"Id"},
	})
	if err != nil || len(got) != 2 || got[1].ID() != 2 {
		t.Fatal(got, err)
	}
	q := srv.Calls[0].Query
	if q.Get("$select") != "Id,Title" || q.Get("$filter") != "Title ne 'x'" || q.Get("$top") != "1" || q.Get("$orderby") != "Id" {
		t.Fatal(q)
	}
}

func TestItemWrites(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"Id":7,"Title":"new"}`))
	})
	s := items.New(srv.Client("/_api"))
	ctx := context.Background()
	l := spx.ByTitle("L")
	it, err := s.Create(ctx, l, map[string]any{"Title": "new"})
	if err != nil || it.ID() != 7 || srv.Last().Body != `{"Title":"new"}` {
		t.Fatal(it, err, srv.Last().Body)
	}
	if err := s.Update(ctx, l, 7, map[string]any{"Title": "x"}); err != nil {
		t.Fatal(err)
	}
	c := srv.Last()
	if c.Path != "/_api/web/lists/getbytitle('L')/items(7)" || c.Header.Get("X-HTTP-Method") != "MERGE" || c.Body != `{"Title":"x"}` {
		t.Fatalf("%+v", c)
	}
	if err := s.Delete(ctx, l, 7); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
	if got, err := s.Get(ctx, l, 7, httpx.ODataQuery{}); err != nil || got.ID() != 7 {
		t.Fatal(got, err)
	}
}
