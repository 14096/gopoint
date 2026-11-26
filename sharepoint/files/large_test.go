package files_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/files"
)

func TestUploadLargeChunks(t *testing.T) {
	oldC, oldS := files.ChunkSize, files.SimpleUploadLimit
	files.ChunkSize, files.SimpleUploadLimit = 4, 5
	defer func() { files.ChunkSize, files.SimpleUploadLimit = oldC, oldS }()

	var got bytes.Buffer
	var ops []string
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case strings.Contains(c.Path, "/Files/add("):
			w.Write([]byte(`{"Name":"b.bin","ServerRelativeUrl":"/sites/t/Docs/b.bin"}`))
		default:
			i := strings.LastIndex(c.Path, "/")
			ops = append(ops, c.Path[i+1:])
			got.WriteString(c.Body)
			w.Write([]byte(`{"Name":"b.bin","ServerRelativeUrl":"/sites/t/Docs/b.bin"}`))
		}
	})
	var progress []int64
	svc := files.New(srv.Client("/_api"))
	f, err := svc.UploadLarge(context.Background(), "/sites/t/Docs", "b.bin", strings.NewReader("0123456789ABC"), 13, true, func(s, _ int64) { progress = append(progress, s) })
	if err != nil || f.Name != "b.bin" {
		t.Fatal(f, err)
	}
	if got.String() != "0123456789ABC" {
		t.Errorf("content = %q", got.String())
	}
	if len(ops) != 4 || !strings.HasPrefix(ops[0], "startupload(") || !strings.Contains(ops[1], "fileOffset=4") ||
		!strings.Contains(ops[3], "finishupload(") || !strings.Contains(ops[3], "fileOffset=12") {
		t.Errorf("ops = %v", ops)
	}
	if len(progress) != 4 || progress[3] != 13 {
		t.Errorf("progress = %v", progress)
	}
}
