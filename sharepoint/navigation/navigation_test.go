package navigation_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/navigation"
)

func TestNavigation(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"value":[{"Id":2,"Title":"Docs"}],"Id":2,"Title":"Docs"}`))
	})
	s := navigation.New(srv.Client("/_api"))
	ctx := context.Background()

	if ns, err := s.All(ctx, navigation.QuickLaunch); err != nil || ns[0].ID != 2 || srv.Last().Path != "/_api/web/navigation/quicklaunch" {
		t.Fatal(ns, err)
	}
	if _, err := s.Add(ctx, navigation.TopBar, navigation.NodeParams{Title: "A", URL: "/a"}); err != nil ||
		srv.Last().Path != "/_api/web/navigation/topnavigationbar" || srv.Last().Body != `{"IsExternal":false,"Title":"A","Url":"/a"}` {
		t.Fatal(err, srv.Last().Body)
	}
	if _, err := s.AddChild(ctx, 2, navigation.NodeParams{Title: "B"}); err != nil || srv.Last().Path != "/_api/web/navigation/GetNodeById(2)/children" {
		t.Fatal(err)
	}
	if cs, err := s.Children(ctx, 2); err != nil || len(cs) != 1 {
		t.Fatal(cs, err)
	}
	if err := s.Update(ctx, 2, navigation.NodeParams{Title: "C"}); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "MERGE" {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, 2); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
}
