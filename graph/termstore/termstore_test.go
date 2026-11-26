package termstore_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/14096/gopoint/graph/termstore"
	"github.com/14096/gopoint/internal/testutil"
)

func TestTermsAndCreate(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method == "POST" {
			w.Write([]byte(`{"id":"t2","labels":[{"name":"Kid","languageTag":"en-US","isDefault":true}]}`))
			return
		}
		w.Write([]byte(`{"value":[{"id":"t1","labels":[{"name":"Root","languageTag":"en-US","isDefault":true}]}]}`))
	})
	svc := termstore.New(srv.Client(""))
	ctx := context.Background()
	n := 0
	for tm, err := range svc.Terms(ctx, "site", "set1") {
		if err != nil || tm.Labels[0].Name != "Root" {
			t.Fatal(tm, err)
		}
		n++
	}
	if n != 1 || srv.Last().Path != "/sites/site/termStore/sets/set1/children" {
		t.Fatal(n, srv.Last().Path)
	}
	tm, err := svc.CreateTerm(ctx, "site", "set1", "t1", []termstore.Label{{Name: "Kid", LanguageTag: "en-US", IsDefault: true}})
	if err != nil || tm.ID != "t2" || !strings.HasSuffix(srv.Last().Path, "/terms/t1/children") {
		t.Fatal(tm, err, srv.Last())
	}
}
