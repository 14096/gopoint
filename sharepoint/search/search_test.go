package search_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/search"
)

const page1 = `{"PrimaryQueryResult":{"RelevantResults":{"RowCount":1,"TotalRows":2,"Table":{"Rows":[{"Cells":[{"Key":"Title","Value":"A"},{"Key":"Path","Value":"/a"}]}]}}}}`
const page2 = `{"PrimaryQueryResult":{"RelevantResults":{"RowCount":1,"TotalRows":2,"Table":{"Rows":[{"Cells":[{"Key":"Title","Value":"B"}]}]}}}}`

func TestSearch(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case c.Path == "/_api/search/suggest":
			w.Write([]byte(`{"Queries":[{"Query":"budget"}]}`))
		case c.Query.Get("startrow") == "1":
			w.Write([]byte(page2))
		default:
			w.Write([]byte(page1))
		}
	})
	s := search.New(srv.Client("/_api"))
	ctx := context.Background()

	res, err := s.Do(ctx, search.Query{Text: "it's", SelectProperties: []string{"Title", "Path"}, RowLimit: 1,
		SortList: []search.Sort{{Property: "Rank", Descending: true}}})
	if err != nil || res.TotalRows != 2 || res.Rows[0]["Path"] != "/a" {
		t.Fatal(res, err)
	}
	q := srv.Last().Query
	if q.Get("querytext") != "'it''s'" || q.Get("selectproperties") != "'Title,Path'" || q.Get("sortlist") != "'Rank:1'" || q.Get("rowlimit") != "1" {
		t.Fatal(q)
	}
	var titles []string
	for r, err := range s.Iter(ctx, search.Query{Text: "x", RowLimit: 1}) {
		if err != nil {
			t.Fatal(err)
		}
		titles = append(titles, r["Title"])
	}
	if len(titles) != 2 || titles[1] != "B" {
		t.Fatal(titles)
	}
	if sg, err := s.Suggest(ctx, "bu"); err != nil || sg[0] != "budget" {
		t.Fatal(sg, err)
	}
}
