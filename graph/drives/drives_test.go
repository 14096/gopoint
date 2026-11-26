package drives_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/graph/drives"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
)

func TestDrivesBasics(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case strings.HasSuffix(c.Path, "/drives"):
			w.Write([]byte(`{"value":[{"id":"d1","name":"Documents"}]}`))
		case strings.HasSuffix(c.Path, "/children"):
			w.Write([]byte(`{"value":[{"id":"f","name":"dir","folder":{"childCount":1}},{"id":"x","name":"a.txt","file":{"mimeType":"text/plain"}}]}`))
		case strings.HasSuffix(c.Path, "/content") && c.Method == "GET":
			w.Write([]byte("data"))
		case c.Method == "DELETE":
			w.WriteHeader(204)
		default:
			w.Write([]byte(`{"id":"d1","name":"Documents","driveType":"documentLibrary"}`))
		}
	})
	s := drives.New(srv.Client("/v1.0"))
	ctx := context.Background()
	if ds, err := s.ForSite(ctx, "s1"); err != nil || len(ds) != 1 {
		t.Fatal(ds, err)
	}
	var names []string
	var folders int
	for it, err := range s.Children(ctx, "d1", "root") {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, it.Name)
		if it.IsFolder() {
			folders++
		}
	}
	if len(names) != 2 || folders != 1 {
		t.Fatal(names, folders)
	}
	if b, err := s.Download(ctx, "d1", "x"); err != nil || string(b) != "data" {
		t.Fatal(string(b), err)
	}
	if _, err := s.GetItemByPath(ctx, "d1", "/a b/c.txt"); err != nil || srv.Last().Path != "/v1.0/drives/d1/root:/a b/c.txt" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.Delete(ctx, "d1", "x"); err != nil {
		t.Fatal(err)
	}
}

func TestSimpleUpload(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"id":"n","name":"a.txt"}`))
	})
	s := drives.New(srv.Client("/v1.0"))
	data := []byte("hello")
	it, err := s.Upload(context.Background(), "d1", "root", "a.txt", bytes.NewReader(data), int64(len(data)))
	c := srv.Last()
	if err != nil || it.ID != "n" || c.Method != "PUT" || c.Path != "/v1.0/drives/d1/items/root:/a.txt:/content" || c.Body != "hello" {
		t.Fatal(it, err, c)
	}
}

func TestLargeUploadUsesSession(t *testing.T) {
	oldLimit, oldChunk := drives.SimpleUploadLimit, drives.ChunkSize
	drives.SimpleUploadLimit, drives.ChunkSize = 10, 8
	defer func() { drives.SimpleUploadLimit, drives.ChunkSize = oldLimit, oldChunk }()

	var received []byte
	var ranges []string
	var srv *testutil.Server
	srv = testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case strings.HasSuffix(c.Path, ":/createUploadSession"):
			if c.Header.Get("Authorization") != "Bearer tok" {
				t.Error("session creation must be authenticated")
			}
			fmt.Fprintf(w, `{"uploadUrl":"%s/upload/abc"}`, srv.URL)
		case c.Path == "/upload/abc":
			if c.Header.Get("Authorization") != "" {
				t.Error("session URL must not carry credentials")
			}
			ranges = append(ranges, c.Header.Get("Content-Range"))
			received = append(received, c.Body...)
			if strings.HasSuffix(c.Header.Get("Content-Range"), "-19/20") {
				w.Write([]byte(`{"id":"big","name":"big.bin"}`))
				return
			}
			w.WriteHeader(202)
			w.Write([]byte(`{"nextExpectedRanges":[]}`))
		default:
			t.Errorf("unexpected %s", c.Path)
		}
	})
	data := []byte("0123456789abcdefghij")
	raw, err := httpx.New(httpx.Config{
		BaseURL: srv.URL + "/v1.0", MaxAttempts: 1,
		Token: func(context.Context) (string, error) { return "tok", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	s := drives.New(raw)
	it, err := s.Upload(context.Background(), "d1", "root", "big.bin", bytes.NewReader(data), int64(len(data)))
	if err != nil || it.ID != "big" {
		t.Fatal(it, err)
	}
	if !bytes.Equal(received, data) {
		t.Fatalf("received %q", received)
	}
	if strings.Join(ranges, ",") != "bytes 0-7/20,bytes 8-15/20,bytes 16-19/20" {
		t.Fatal(ranges)
	}
}
