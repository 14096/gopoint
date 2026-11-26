package search_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/graph/search"
	"github.com/14096/gopoint/internal/testutil"
)

func TestSearch(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Body == "" {
			t.Error("empty body")
		}
		if strings.Contains(c.Body, `"from":1`) {
			w.Write([]byte(`{"value":[{"hitsContainers":[{"total":2,"moreResultsAvailable":false,"hits":[{"hitId":"b","rank":2}]}]}]}`))
			return
		}
		w.Write([]byte(`{"value":[{"hitsContainers":[{"total":2,"moreResultsAvailable":true,"hits":[{"hitId":"a","rank":1,"summary":"s","resource":{"name":"x"}}]}]}]}`))
	})
	s := search.New(srv.Client("/v1.0"))
	ctx := context.Background()

	if _, err := s.Do(ctx, search.Query{}); err == nil {
		t.Fatal("expected validation error")
	}
	res, err := s.Do(ctx, search.Query{EntityTypes: []string{"driveItem"}, Text: "budget", Size: 1, Region: "EUR"})
	if err != nil || res.Total != 2 || !res.MoreAvailable || res.Hits[0].ID != "a" || string(res.Hits[0].Resource) != `{"name":"x"}` {
		t.Fatal(res, err)
	}
	if srv.Last().Path != "/v1.0/search/query" || srv.Last().Body != `{"requests":[{"entityTypes":["driveItem"],"from":0,"query":{"queryString":"budget"},"region":"EUR","size":1}]}` {
		t.Fatal(srv.Last().Body)
	}
	var r struct{ Name string }
	if err := res.Hits[0].Decode(&r); err != nil || r.Name != "x" {
		t.Fatal(r, err)
	}
	var ids []string
	for h, err := range s.Iter(ctx, search.Query{EntityTypes: []string{"driveItem"}, Text: "x"}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, h.ID)
	}
	if len(ids) != 2 || ids[1] != "b" {
		t.Fatal(ids)
	}
}
