package webhooks_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/spx"
	"github.com/14096/gopoint/sharepoint/webhooks"
)

func TestAddRenewDelete(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"id":"s1","clientState":"cs"}`))
	})
	svc := webhooks.New(srv.Client("/sites/t/_api"), "https://x.sharepoint.com/sites/t")
	ctx := context.Background()
	exp := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	s, err := svc.Add(ctx, spx.ByID("L1"), "https://hook", exp, "cs")
	if err != nil || s.ID != "s1" {
		t.Fatal(s, err)
	}
	c := srv.Last()
	if c.Method != "POST" || !strings.HasSuffix(c.Path, "/subscriptions") ||
		!strings.Contains(c.Body, `"resource":"https://x.sharepoint.com/sites/t/_api/web/lists(guid'L1')"`) ||
		!strings.Contains(c.Body, "2027-01-02T03:04:05.0000000Z") {
		t.Errorf("%+v", c)
	}
	if err := svc.Renew(ctx, spx.ByID("L1"), "s1", exp); err != nil || srv.Last().Method != "PATCH" {
		t.Fatal(err, srv.Last())
	}
	if err := svc.Delete(ctx, spx.ByID("L1"), "s1"); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
}

func TestHandler(t *testing.T) {
	var got []webhooks.Notification
	h := webhooks.Handler("cs", func(n webhooks.Notification) { got = append(got, n) })
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/?validationtoken=abc", nil))
	if rec.Body.String() != "abc" || rec.Code != 200 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(
		`{"value":[{"subscriptionId":"a","clientState":"cs"},{"subscriptionId":"b","clientState":"evil"}]}`)))
	if rec.Code != 200 || len(got) != 1 || got[0].SubscriptionID != "a" {
		t.Fatal(rec.Code, got)
	}
}
