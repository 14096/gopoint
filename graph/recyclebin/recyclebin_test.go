package recyclebin_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/recyclebin"
	"github.com/14096/gopoint/internal/testutil"
)

func TestRecycleBin(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method == "POST" {
			w.WriteHeader(204)
			return
		}
		w.Write([]byte(`{"value":[{"id":"r1","name":"a.txt","size":3}]}`))
	})
	s := recyclebin.New(srv.Client("/beta"))
	ctx := context.Background()
	n := 0
	for it, err := range s.Iter(ctx, "s") {
		if err != nil || it.ID != "r1" {
			t.Fatal(it, err)
		}
		n++
	}
	if n != 1 || srv.Last().Path != "/beta/sites/s/recycleBin/items" {
		t.Fatal(n, srv.Last().Path)
	}
	if is, err := s.All(ctx, "s"); err != nil || len(is) != 1 {
		t.Fatal(is, err)
	}
	if err := s.Restore(ctx, "s", "r1", "r2"); err != nil || srv.Last().Path != "/beta/sites/s/recycleBin/items/restore" || srv.Last().Body != `{"ids":["r1","r2"]}` {
		t.Fatal(err, srv.Last().Body)
	}
	if err := s.Delete(ctx, "s", "r1"); err != nil || srv.Last().Path != "/beta/sites/s/recycleBin/items/delete" {
		t.Fatal(err)
	}
}
