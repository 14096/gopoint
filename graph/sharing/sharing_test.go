package sharing_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/permissions"
	"github.com/14096/gopoint/graph/sharing"
	"github.com/14096/gopoint/internal/testutil"
)

func TestSharing(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		switch {
		case c.Method == "DELETE":
			w.WriteHeader(204)
		case c.Path == "/v1.0/drives/d/items/i/invite":
			w.Write([]byte(`{"value":[{"id":"p","roles":["write"]}]}`))
		case c.Path == "/v1.0/drives/d/items/i/createLink":
			w.Write([]byte(`{"id":"l","link":{"type":"view","scope":"anonymous","webUrl":"https://l"}}`))
		default:
			w.Write([]byte(`{"id":"x","name":"a.txt"}`))
		}
	})
	s := sharing.New(srv.Client("/v1.0"))
	ctx := context.Background()

	p, err := s.CreateLink(ctx, "d", "i", sharing.LinkParams{Scope: sharing.AnyoneScope})
	if err != nil || p.Link.WebURL != "https://l" || srv.Last().Body != `{"scope":"anonymous","type":"view"}` {
		t.Fatal(p, err, srv.Last().Body)
	}
	ps, err := s.Invite(ctx, "d", "i", sharing.InviteParams{Emails: []string{"a@b.c"}, Roles: []permissions.Role{permissions.Write}})
	if err != nil || ps[0].ID != "p" || srv.Last().Body != `{"recipients":[{"email":"a@b.c"}],"requireSignIn":false,"roles":["write"],"sendInvitation":false}` {
		t.Fatal(ps, err, srv.Last().Body)
	}
	if _, err := s.Invite(ctx, "d", "i", sharing.InviteParams{}); err == nil {
		t.Fatal("expected error")
	}
	if err := s.Unshare(ctx, "d", "i", "l"); err != nil || srv.Last().Path != "/v1.0/drives/d/items/i/permissions/l" {
		t.Fatal(err)
	}
	if it, err := s.Resolve(ctx, "https://x/a?b=1~"); err != nil || it.ID != "x" || srv.Last().Path != "/v1.0/shares/u!aHR0cHM6Ly94L2E_Yj0xfg/driveItem" {
		t.Fatal(it, err, srv.Last().Path)
	}
}
