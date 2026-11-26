package fields_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/fields"
	"github.com/14096/gopoint/sharepoint/spx"
)

func TestFields(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method == "GET" {
			w.Write([]byte(`{"value":[{"InternalName":"Title","TypeAsString":"Text"}]}`))
			return
		}
		w.Write([]byte(`{"InternalName":"Budget","StaticName":"Budget","FieldTypeKind":10}`))
	})
	s := fields.New(srv.Client("/_api"))
	ctx := context.Background()
	list := spx.ByTitle("Projects")

	got, err := s.List(ctx, list, httpx.ODataQuery{Filter: "Hidden eq false"})
	if err != nil || len(got) != 1 || srv.Last().Path != "/_api/web/lists/getbytitle('Projects')/fields" {
		t.Fatal(got, err, srv.Last().Path)
	}
	if srv.Last().Query["$filter"][0] != "Hidden eq false" {
		t.Fatal(srv.Last().Query)
	}
	if _, err := s.List(ctx, spx.Web, httpx.ODataQuery{}); err != nil || srv.Last().Path != "/_api/web/fields" {
		t.Fatal(err, srv.Last().Path)
	}
	f, err := s.Create(ctx, list, fields.CreateParams{Title: "Budget", StaticName: "Budget", FieldTypeKind: fields.Currency, Extra: map[string]any{"MinimumValue": 0}})
	if err != nil || f.InternalName != "Budget" {
		t.Fatal(f, err)
	}
	var body map[string]any
	json.Unmarshal([]byte(srv.Last().Body), &body)
	if body["FieldTypeKind"] != float64(10) || body["StaticName"] != "Budget" || body["MinimumValue"] != float64(0) {
		t.Fatal(body)
	}
	if err := s.Update(ctx, list, "Budget", map[string]any{"Required": true}); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "MERGE" {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, list, "Budget"); err != nil || srv.Last().Path != "/_api/web/lists/getbytitle('Projects')/fields/getbyinternalnameortitle('Budget')" {
		t.Fatal(err, srv.Last().Path)
	}
}
