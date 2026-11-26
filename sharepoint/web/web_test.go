package web_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/web"
)

func TestGetSubwebsCreate(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch c.Path {
		case "/_api/web":
			w.Write([]byte(`{"Id":"g1","Title":"Team","Url":"https://x/sites/t","Description":"d"}`))
		case "/_api/web/webs":
			w.Write([]byte(`{"value":[{"Title":"a"},{"Title":"b"}]}`))
		case "/_api/web/webinfos/add":
			w.Write([]byte(`{"Title":"New","Url":"https://x/sites/t/new"}`))
		}
	})
	s := web.New(srv.Client("/_api"))
	ctx := context.Background()
	w, err := s.Get(ctx)
	if err != nil || w.Title != "Team" || w.ID != "g1" || w.URL == "" || w.Description != "d" {
		t.Fatal(w, err)
	}
	subs, err := s.Subwebs(ctx)
	if err != nil || len(subs) != 2 {
		t.Fatal(subs, err)
	}
	n, err := s.CreateSubweb(ctx, web.CreateParams{Title: "New", URL: "new"})
	if err != nil || n.Title != "New" {
		t.Fatal(n, err)
	}
	if c := srv.Last(); c.Method != "POST" || c.Body == "" {
		t.Fatalf("%+v", c)
	}
}
