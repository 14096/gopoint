package sharing_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/sharing"
	"github.com/14096/gopoint/sharepoint/spx"
)

func TestSharing(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if strings.HasSuffix(c.Path, "/ShareLink") {
			w.Write([]byte(`{"sharingLinkInfo":{"Url":"https://x/l","LinkKind":4,"ShareId":"g","IsAnonymous":true}}`))
			return
		}
		w.Write([]byte(`{"value":[{"User":"a@b.c","Status":true}]}`))
	})
	s := sharing.New(srv.Client("/_api"))
	ctx := context.Background()
	f := spx.FileItem("/sites/t/D/a.txt")

	l, err := s.CreateLink(ctx, f, sharing.LinkParams{Kind: sharing.AnonymousView, Expires: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)})
	if err != nil || l.URL != "https://x/l" || !l.IsAnonymous ||
		srv.Last().Path != "/_api/web/GetFileByServerRelativeUrl('/sites/t/D/a.txt')/ListItemAllFields/ShareLink" ||
		srv.Last().Body != `{"request":{"createLink":true,"settings":{"expiration":"2030-01-02T03:04:05Z","linkKind":4}}}` {
		t.Fatal(l, err, srv.Last().Body)
	}
	if _, err := s.CreateLink(ctx, f, sharing.LinkParams{}); err == nil {
		t.Fatal("expected error for missing kind")
	}
	if err := s.DeleteLink(ctx, f, sharing.AnonymousView); err != nil || !strings.HasSuffix(srv.Last().Path, "/UnshareLink") {
		t.Fatal(err)
	}
	rs, err := s.Invite(ctx, sharing.InviteParams{
		ResourceURL: "https://x/sites/t/D/a.txt",
		Recipients:  []sharing.Recipient{{Email: "a@b.c", Role: sharing.Edit}, {Email: "d@e.f"}},
	})
	if err != nil || !rs[0].Status || !strings.Contains(srv.Last().Body, `"UserId":"d@e.f"`) || !strings.Contains(srv.Last().Body, `"Role":2`) {
		t.Fatal(rs, err, srv.Last().Body)
	}
	if _, err := s.Invite(ctx, sharing.InviteParams{}); err == nil {
		t.Fatal("expected error")
	}
}
