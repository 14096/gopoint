package contenttypes_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/contenttypes"
	"github.com/14096/gopoint/sharepoint/spx"
)

func TestContentTypes(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method == "GET" && c.Path == "/_api/web/lists/getbytitle('L')/contenttypes" {
			w.Write([]byte(`{"value":[{"StringId":"0x01","Name":"Item"},{"StringId":"0x0120","Name":"Folder"}]}`))
			return
		}
		w.Write([]byte(`{"StringId":"0x0100AB","Name":"Mine"}`))
	})
	s := contenttypes.New(srv.Client("/_api"))
	ctx := context.Background()
	got, err := s.List(ctx, spx.ByTitle("L"))
	if err != nil || len(got) != 2 || got[0].ID != "0x01" || got[1].Name != "Folder" {
		t.Fatal(got, err)
	}
	ct, err := s.Create(ctx, spx.Web, contenttypes.CreateParams{ID: "0x0100AB", Name: "Mine"})
	if err != nil || ct.Name != "Mine" || srv.Last().Path != "/_api/web/contenttypes" {
		t.Fatal(ct, err)
	}
	if _, err := s.Get(ctx, spx.Web, "0x0100AB"); err != nil || srv.Last().Path != "/_api/web/contenttypes('0x0100AB')" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.Delete(ctx, spx.Web, "0x0100AB"); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
}
