package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type item struct{ N int }

func TestPaginateGraphStyle(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("p") {
		case "":
			fmt.Fprintf(w, `{"value":[{"N":1},{"N":2}],"@odata.nextLink":"%s/api/c?p=2"}`, srv.URL)
		case "2":
			fmt.Fprintf(w, `{"value":[{"N":3}],"@odata.nextLink":"%s/api/c?p=3"}`, srv.URL)
		default:
			fmt.Fprint(w, `{"value":[{"N":4}]}`)
		}
	}))
	defer srv.Close()
	c, err := New(Config{BaseURL: srv.URL + "/api"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Collect[item](context.Background(), c, NewRequest("GET", "c"))
	if err != nil || len(got) != 4 || got[3].N != 4 {
		t.Fatal(got, err)
	}
}

func TestPaginateSharePointVerbose(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("p") == "" {
			fmt.Fprintf(w, `{"d":{"results":[{"N":1}],"__next":"%s/api/c?p=2"}}`, srv.URL)
			return
		}
		fmt.Fprint(w, `{"d":{"results":[{"N":2}]}}`)
	}))
	defer srv.Close()
	c, err := New(Config{BaseURL: srv.URL + "/api"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Collect[item](context.Background(), c, NewRequest("GET", "c"))
	if err != nil || len(got) != 2 {
		t.Fatal(got, err)
	}
}

func TestPaginateCancel(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"value":[{"N":1}],"odata.nextLink":"%s/api/c"}`, srv.URL)
	}))
	defer srv.Close()
	c, err := New(Config{BaseURL: srv.URL + "/api"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var last error
	n := 0
	for _, err := range Paginate[item](ctx, c, NewRequest("GET", "c")) {
		last = err
		if n++; n == 2 {
			cancel()
		}
	}
	if !errors.Is(last, context.Canceled) {
		t.Fatal(last)
	}
}
