package sharepoint_test

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint"
)

func TestBatchRoundTrip(t *testing.T) {
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/contextinfo") {
			w.Write([]byte(`{"FormDigestValue":"d","FormDigestTimeoutSeconds":1800}`))
			return
		}
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}
			b, _ := io.ReadAll(p)
			sent = append(sent, p.Header.Get("Content-Type")+"|"+string(b))
		}
		w.Header().Set("Content-Type", "multipart/mixed; boundary=resp")
		w.Write([]byte("--resp\r\nContent-Type: application/http\r\nContent-Transfer-Encoding: binary\r\n\r\n" +
			"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"Title\":\"T\"}\r\n" +
			"--resp\r\nContent-Type: multipart/mixed; boundary=cs\r\n\r\n" +
			"--cs\r\nContent-Type: application/http\r\nContent-Transfer-Encoding: binary\r\n\r\n" +
			"HTTP/1.1 404 Not Found\r\nContent-Type: application/json\r\n\r\n{\"odata.error\":{\"code\":\"x\",\"message\":{\"value\":\"nope\"}}}\r\n" +
			"--cs--\r\n--resp--\r\n"))
	}))
	defer srv.Close()
	c, err := sharepoint.New(srv.URL+"/sites/t", &fakeCred{}, sharepoint.Options{MaxAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Batch(context.Background(), []sharepoint.BatchRequest{
		{Path: "web?$select=Title"},
		{Method: "POST", Path: "web/lists", Body: map[string]string{"Title": "x"}},
	})
	if err != nil || len(res) != 2 {
		t.Fatal(res, err)
	}
	var web struct{ Title string }
	if err := res[0].JSON(&web); err != nil || web.Title != "T" || res[0].Err() != nil {
		t.Fatal(web, err)
	}
	var he *httpx.Error
	if e := res[1].Err(); e == nil || !httpx.IsNotFound(e) {
		t.Fatal(e, he)
	}
	if len(sent) != 2 || !strings.Contains(sent[0], "GET "+srv.URL+"/sites/t/_api/web?$select=Title HTTP/1.1") ||
		!strings.HasPrefix(sent[1], "multipart/mixed") || !strings.Contains(sent[1], `{"Title":"x"}`) {
		t.Fatalf("sent = %q", sent)
	}
}
