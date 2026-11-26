package subscriptions_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/14096/gopoint/graph/subscriptions"
	"github.com/14096/gopoint/internal/testutil"
)

func TestLifecycle(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) { w.Write([]byte(`{"id":"s1"}`)) })
	svc := subscriptions.New(srv.Client(""))
	ctx := context.Background()
	if s, err := svc.Create(ctx, subscriptions.Subscription{Resource: "/sites/x/lists/y", ChangeType: "updated"}); err != nil || s.ID != "s1" {
		t.Fatal(s, err)
	}
	if srv.Last().Method != "POST" || srv.Last().Path != "/subscriptions" {
		t.Errorf("%+v", srv.Last())
	}
	if _, err := svc.Renew(ctx, "s1", time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil ||
		srv.Last().Method != "PATCH" || !strings.Contains(srv.Last().Body, "2027-01-01T00:00:00Z") {
		t.Fatal(err, srv.Last())
	}
	if err := svc.Delete(ctx, "s1"); err != nil || srv.Last().Method != "DELETE" {
		t.Fatal(err)
	}
}

func TestHandler(t *testing.T) {
	var got []subscriptions.Notification
	h := subscriptions.Handler("cs", func(n subscriptions.Notification) { got = append(got, n) })
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/?validationToken=tok", nil))
	if rec.Body.String() != "tok" {
		t.Fatal(rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(`{"value":[{"subscriptionId":"a","clientState":"cs"},{"subscriptionId":"b"}]}`)))
	if rec.Code != 202 || len(got) != 1 {
		t.Fatal(rec.Code, got)
	}
}
