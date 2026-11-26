package files_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/files"
)

func TestUploadDownloadRoundTrip(t *testing.T) {
	store := map[string]string{}
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case strings.Contains(c.Path, "/Files/add("):
			store["/sites/t/Docs/a b.txt"] = c.Body
			w.Write([]byte(`{"Name":"a b.txt","ServerRelativeUrl":"/sites/t/Docs/a b.txt","Length":"5"}`))
		case strings.HasSuffix(c.Path, "/$value"):
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte(store["/sites/t/Docs/a b.txt"]))
		case strings.HasSuffix(c.Path, "/Files"):
			w.Write([]byte(`{"value":[{"Name":"a b.txt","Length":"5"}]}`))
		case strings.HasSuffix(c.Path, "/Folders"):
			w.Write([]byte(`{"value":[{"Name":"sub"}]}`))
		default:
			w.Write([]byte(`{"Name":"new","ServerRelativeUrl":"/sites/t/Docs/new"}`))
		}
	})
	s := files.New(srv.Client("/_api"))
	ctx := context.Background()

	f, err := s.Upload(ctx, "/sites/t/Docs", "a b.txt", []byte("hello"), true)
	if err != nil || f.Length.String() != "5" {
		t.Fatal(f, err)
	}
	if p := srv.Last().Path; p != "/_api/web/GetFolderByServerRelativeUrl('/sites/t/Docs')/Files/add(url='a b.txt',overwrite=true)" {
		t.Fatal(p)
	}
	got, err := s.Download(ctx, "/sites/t/Docs/a b.txt")
	if err != nil || string(got) != "hello" {
		t.Fatalf("%q %v", got, err)
	}
	if fs, err := s.ListFiles(ctx, "/sites/t/Docs"); err != nil || len(fs) != 1 {
		t.Fatal(fs, err)
	}
	if fs, err := s.ListFolders(ctx, "/sites/t/Docs"); err != nil || fs[0].Name != "sub" {
		t.Fatal(fs, err)
	}
	if fo, err := s.CreateFolder(ctx, "/sites/t/Docs/new"); err != nil || fo.Name != "new" {
		t.Fatal(fo, err)
	}
	if err := s.DeleteFile(ctx, "/sites/t/Docs/a b.txt"); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
	if err := s.DeleteFolder(ctx, "/sites/t/Docs/new"); err != nil {
		t.Fatal(err)
	}
}
