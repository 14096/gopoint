package folders_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/folders"
)

func TestFolders(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if strings.HasSuffix(c.Path, "/Exists") {
			w.Write([]byte(`{"value":true}`))
			return
		}
		w.Write([]byte(`{"value":[{"Name":"sub"}],"Name":"Docs"}`))
	})
	s := folders.New(srv.Client("/_api"))
	ctx := context.Background()
	const f = "/sites/t/Docs"
	const p = "/_api/web/GetFolderByServerRelativeUrl('/sites/t/Docs')"

	if g, err := s.Get(ctx, f); err != nil || g.Name != "Docs" || srv.Last().Path != p {
		t.Fatal(g, err)
	}
	if ok, err := s.Exists(ctx, f); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if cs, err := s.All(ctx, f); err != nil || cs[0].Name != "sub" {
		t.Fatal(cs, err)
	}
	n := 0
	for _, err := range s.Iter(ctx, f) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
	if _, err := s.Create(ctx, f+"/new"); err != nil || srv.Last().Method != "POST" {
		t.Fatal(err)
	}
	if err := s.Rename(ctx, f, "Renamed"); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "MERGE" || srv.Last().Path != p+"/ListItemAllFields" {
		t.Fatal(err)
	}
	if err := s.Move(ctx, f, "/sites/t/Other"); err != nil || srv.Last().Path != p+"/moveto(newurl='/sites/t/Other',flags=1)" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.Delete(ctx, f); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
}
