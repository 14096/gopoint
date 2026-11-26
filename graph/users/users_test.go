package users_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/graph/users"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
)

func TestUsers(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		if c.Method != "GET" {
			w.WriteHeader(204)
			return
		}
		w.Write([]byte(`{"value":[{"id":"u1","displayName":"A"}],"id":"u1","displayName":"A","mail":"a@b.c"}`))
	})
	s := users.New(srv.Client("/v1.0"))
	ctx := context.Background()

	if u, err := s.Me(ctx); err != nil || u.Mail != "a@b.c" || srv.Last().Path != "/v1.0/me" {
		t.Fatal(u, err)
	}
	if u, err := s.Get(ctx, "a@b.c"); err != nil || u.ID != "u1" {
		t.Fatal(u, err)
	}
	for u, err := range s.Iter(ctx, httpx.ODataQuery{Top: 5}) {
		if err != nil || u.ID != "u1" {
			t.Fatal(u, err)
		}
	}
	if srv.Last().Query.Get("$top") != "5" {
		t.Fatal(srv.Last().Query)
	}
	if us, err := s.All(ctx, httpx.ODataQuery{}); err != nil || len(us) != 1 {
		t.Fatal(us, err)
	}
	for _, err := range s.Groups(ctx, httpx.ODataQuery{}) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if g, err := s.GetGroup(ctx, "g"); err != nil || g.ID != "u1" {
		t.Fatal(g, err)
	}
	if gs, err := s.MemberOf(ctx, "u1"); err != nil || len(gs) != 1 || srv.Last().Path != "/v1.0/users/u1/memberOf/microsoft.graph.group" {
		t.Fatal(gs, err)
	}
	if us, err := s.Members(ctx, "g"); err != nil || len(us) != 1 {
		t.Fatal(us, err)
	}
	if err := s.AddMember(ctx, "g", "u1"); err != nil || srv.Last().Path != "/v1.0/groups/g/members/$ref" ||
		srv.Last().Body != `{"@odata.id":"https://graph.microsoft.com/v1.0/directoryObjects/u1"}` {
		t.Fatal(err, srv.Last().Body)
	}
	if err := s.RemoveMember(ctx, "g", "u1"); err != nil || srv.Last().Method != "DELETE" || srv.Last().Path != "/v1.0/groups/g/members/u1/$ref" {
		t.Fatal(err)
	}
}
