package recyclebin_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/recyclebin"
)

func TestRecycleBin(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"value":[{"Id":"abc","LeafName":"a.txt","Size":"12"}],"Id":"abc","LeafName":"a.txt"}`))
	})
	s := recyclebin.New(srv.Client("/_api"))
	ctx := context.Background()
	const it = "/_api/web/recyclebin(guid'abc')"

	if is, err := s.All(ctx, recyclebin.Site, httpx.ODataQuery{Top: 5}); err != nil || is[0].Size.String() != "12" || srv.Last().Path != "/_api/site/recyclebin" {
		t.Fatal(is, err)
	}
	n := 0
	for _, err := range s.Iter(ctx, recyclebin.Web, httpx.ODataQuery{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
	if i, err := s.Get(ctx, recyclebin.Web, "abc"); err != nil || i.LeafName != "a.txt" || srv.Last().Path != it {
		t.Fatal(i, err)
	}
	for name, call := range map[string]func() error{
		"/restore":           func() error { return s.Restore(ctx, recyclebin.Web, "abc") },
		"/deleteobject":      func() error { return s.DeletePermanently(ctx, recyclebin.Web, "abc") },
		"/movetosecondstage": func() error { return s.MoveToSecondStage(ctx, "abc") },
	} {
		if err := call(); err != nil || srv.Last().Path != it+name {
			t.Fatal(name, err, srv.Last().Path)
		}
	}
	if err := s.RestoreAll(ctx, recyclebin.Site); err != nil || srv.Last().Path != "/_api/site/recyclebin/restoreall" {
		t.Fatal(err)
	}
	if err := s.Empty(ctx, recyclebin.Web); err != nil || srv.Last().Path != "/_api/web/recyclebin/deleteall" {
		t.Fatal(err)
	}
}
