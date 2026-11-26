package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/httpx"
)

type fakeCred struct{}

func (fakeCred) Token(context.Context, auth.TokenOptions) (auth.AccessToken, error) {
	return auth.AccessToken{Token: "t"}, nil
}

func TestBatchChunksAndOrders(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var in struct{ Requests []batchItem }
		json.NewDecoder(r.Body).Decode(&in)
		if len(in.Requests) > MaxBatchSize {
			t.Errorf("chunk of %d", len(in.Requests))
		}
		var rs []map[string]any
		for i := len(in.Requests) - 1; i >= 0; i-- { // reversed on purpose
			q := in.Requests[i]
			st := 200
			if q.URL == "/missing" {
				st = 404
			}
			rs = append(rs, map[string]any{"id": q.ID, "status": st, "body": map[string]string{"url": q.URL}})
		}
		json.NewEncoder(w).Encode(map[string]any{"responses": rs})
	}))
	defer srv.Close()
	c, err := New(fakeCred{}, Options{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	reqs := make([]BatchRequest, 25)
	for i := range reqs {
		reqs[i] = BatchRequest{URL: "/u" + string(rune('a'+i))}
	}
	reqs[3].URL = "/missing"
	res, err := c.Batch(context.Background(), reqs)
	if err != nil || calls != 2 || len(res) != 25 {
		t.Fatal(err, calls, len(res))
	}
	var b struct{ URL string }
	res[24].JSON(&b)
	if b.URL != reqs[24].URL {
		t.Errorf("order: %q", b.URL)
	}
	if !httpx.IsNotFound(res[3].Err()) || res[0].Err() != nil {
		t.Error("error mapping")
	}
}
