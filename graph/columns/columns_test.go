package columns_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/columns"
	"github.com/14096/gopoint/internal/testutil"
)

func TestColumns(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method == "POST" {
			w.Write([]byte(`{"id":"c1","name":"Budget"}`))
			return
		}
		w.Write([]byte(`{"value":[{"id":"c0","name":"Title"}]}`))
	})
	s := columns.New(srv.Client("/v1.0"))
	ctx := context.Background()
	if cs, err := s.List(ctx, "s1", ""); err != nil || len(cs) != 1 || srv.Last().Path != "/v1.0/sites/s1/columns" {
		t.Fatal(cs, err, srv.Last().Path)
	}
	if _, err := s.List(ctx, "s1", "l1"); err != nil || srv.Last().Path != "/v1.0/sites/s1/lists/l1/columns" {
		t.Fatal(err, srv.Last().Path)
	}
	c, err := s.Create(ctx, "s1", "l1", map[string]any{"name": "Budget", "number": map[string]any{}})
	if err != nil || c.Name != "Budget" {
		t.Fatal(c, err)
	}
	if err := s.Delete(ctx, "s1", "l1", "c1"); err != nil || srv.Last().Path != "/v1.0/sites/s1/lists/l1/columns/c1" {
		t.Fatal(err)
	}
}
