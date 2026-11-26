package lists_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/lists"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
)

func TestListsAcrossPages(t *testing.T) {
	var srv *testutil.Server
	srv = testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case c.Method == "POST":
			w.Write([]byte(`{"id":"l1","displayName":"Tasks","list":{"template":"genericList"}}`))
		case c.Query.Get("p") == "":
			fmt.Fprintf(w, `{"value":[{"id":"a"}],"@odata.nextLink":"%s/v1.0/sites/s1/lists?p=2"}`, srv.URL)
		default:
			w.Write([]byte(`{"value":[{"id":"b"}]}`))
		}
	})
	s := lists.New(srv.Client("/v1.0"))
	ctx := context.Background()
	all, err := s.All(ctx, "s1", httpx.ODataQuery{})
	if err != nil || len(all) != 2 {
		t.Fatal(all, err)
	}
	l, err := s.Create(ctx, "s1", lists.CreateParams{DisplayName: "Tasks"})
	if err != nil || l.Info.Template != "genericList" {
		t.Fatal(l, err)
	}
	if srv.Last().Body != `{"displayName":"Tasks","list":{"template":"genericList"}}` {
		t.Fatal(srv.Last().Body)
	}
	if _, err := s.Get(ctx, "s1", "l1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "s1", "l1"); err != nil || srv.Last().Method != "DELETE" {
		t.Fatal(err)
	}
}
