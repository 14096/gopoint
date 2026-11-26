package contenttypes_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/contenttypes"
	"github.com/14096/gopoint/internal/testutil"
)

func TestContentTypes(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Path == "/v1.0/sites/s1/contentTypes" {
			w.Write([]byte(`{"value":[{"id":"0x01","name":"Item"}]}`))
			return
		}
		w.Write([]byte(`{"id":"0x0101","name":"Document"}`))
	})
	s := contenttypes.New(srv.Client("/v1.0"))
	ctx := context.Background()
	got, err := s.List(ctx, "s1", "")
	if err != nil || len(got) != 1 || got[0].Name != "Item" {
		t.Fatal(got, err)
	}
	if _, err := s.List(ctx, "s1", "l1"); err != nil || srv.Last().Path != "/v1.0/sites/s1/lists/l1/contentTypes" {
		t.Fatal(err, srv.Last().Path)
	}
	if ct, err := s.Get(ctx, "s1", "", "0x0101"); err != nil || ct.Name != "Document" {
		t.Fatal(ct, err)
	}
}
