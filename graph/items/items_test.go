package items_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/graph/items"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
)

func TestItems(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case c.Method == "POST":
			w.Write([]byte(`{"id":"5","fields":{"Title":"new"}}`))
		case c.Method == "PATCH":
			w.Write([]byte(`{"Title":"patched"}`))
		case c.Method == "DELETE":
			w.WriteHeader(204)
		default:
			w.Write([]byte(`{"value":[{"id":"1","fields":{"Title":"a"}}]}`))
		}
	})
	s := items.New(srv.Client("/v1.0"))
	ctx := context.Background()

	got, err := s.All(ctx, "s1", "l1", httpx.ODataQuery{Filter: "fields/Title eq 'a'", Top: 5})
	if err != nil || len(got) != 1 || got[0].Fields["Title"] != "a" {
		t.Fatal(got, err)
	}
	q := srv.Last().Query
	if q.Get("$expand") != "fields" || q.Get("$filter") != "fields/Title eq 'a'" || q.Get("$top") != "5" {
		t.Fatal(q)
	}
	it, err := s.Create(ctx, "s1", "l1", map[string]any{"Title": "new"})
	if err != nil || it.ID != "5" || srv.Last().Body != `{"fields":{"Title":"new"}}` {
		t.Fatal(it, err, srv.Last().Body)
	}
	f, err := s.Update(ctx, "s1", "l1", "5", map[string]any{"Title": "patched"})
	if err != nil || f["Title"] != "patched" || srv.Last().Path != "/v1.0/sites/s1/lists/l1/items/5/fields" {
		t.Fatal(f, err, srv.Last().Path)
	}
	if err := s.Delete(ctx, "s1", "l1", "5"); err != nil {
		t.Fatal(err)
	}
}

func TestDelta(t *testing.T) {
	var srvURL string
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Query.Get("token") != "" {
			w.Write([]byte(`{"value":[{"id":"3"}],"@odata.deltaLink":"` + srvURL + `/sites/s/lists/l/items/delta?token=n"}`))
			return
		}
		w.Write([]byte(`{"value":[{"id":"1"},{"id":"2"}],"@odata.deltaLink":"` + srvURL + `/sites/s/lists/l/items/delta?token=a"}`))
	})
	srvURL = srv.URL
	svc := items.New(srv.Client(""))
	got, link, err := svc.Delta(context.Background(), "s", "l", "")
	if err != nil || len(got) != 2 || !strings.HasSuffix(link, "token=a") {
		t.Fatal(got, link, err)
	}
	got, link, err = svc.Delta(context.Background(), "s", "l", link)
	if err != nil || len(got) != 1 || !strings.HasSuffix(link, "token=n") {
		t.Fatal(got, link, err)
	}
}
