package folders_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/folders"
	"github.com/14096/gopoint/internal/testutil"
)

func TestFolders(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Path == "/v1.0/drives/d/items/f/copy" {
			w.Header().Set("Location", "https://mon/1")
			w.WriteHeader(202)
			return
		}
		if c.Method == "DELETE" || c.Path == "/v1.0/drives/d/items/f/permanentDelete" {
			w.WriteHeader(204)
			return
		}
		w.Write([]byte(`{"value":[{"id":"c","name":"child"}],"id":"f","name":"dir","folder":{"childCount":1}}`))
	})
	s := folders.New(srv.Client("/v1.0"))
	ctx := context.Background()

	if f, err := s.Get(ctx, "d", "f"); err != nil || !f.IsFolder() {
		t.Fatal(f, err)
	}
	n := 0
	for _, err := range s.Iter(ctx, "d", "f") {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
	if cs, err := s.All(ctx, "d", "f"); err != nil || len(cs) != 1 {
		t.Fatal(cs, err)
	}
	if _, err := s.Create(ctx, "d", "root", "new"); err != nil || srv.Last().Path != "/v1.0/drives/d/items/root/children" ||
		srv.Last().Body != `{"@microsoft.graph.conflictBehavior":"fail","folder":{},"name":"new"}` {
		t.Fatal(err, srv.Last().Body)
	}
	if _, err := s.Rename(ctx, "d", "f", "n2"); err != nil || srv.Last().Method != "PATCH" || srv.Last().Body != `{"name":"n2"}` {
		t.Fatal(err)
	}
	if _, err := s.Move(ctx, "d", "f", "p", ""); err != nil || srv.Last().Body != `{"parentReference":{"id":"p"}}` {
		t.Fatal(err, srv.Last().Body)
	}
	if loc, err := s.Copy(ctx, "d", "f", "d2", "p", "c"); err != nil || loc != "https://mon/1" {
		t.Fatal(loc, err)
	}
	if err := s.Delete(ctx, "d", "f"); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePermanently(ctx, "d", "f"); err != nil {
		t.Fatal(err)
	}
}
