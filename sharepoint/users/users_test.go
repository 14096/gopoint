package users_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/users"
)

func TestUsersAndGroups(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"value":[{"Id":1,"Title":"A"}],"Id":1,"Title":"A","Email":"a@b.c"}`))
	})
	s := users.New(srv.Client("/_api"))
	ctx := context.Background()

	check := func(path string) {
		t.Helper()
		if srv.Last().Path != "/_api/"+path {
			t.Fatalf("got %s want %s", srv.Last().Path, path)
		}
	}
	if u, err := s.Current(ctx); err != nil || u.Email != "a@b.c" {
		t.Fatal(u, err)
	}
	check("web/currentuser")
	if us, err := s.All(ctx, httpx.ODataQuery{}); err != nil || len(us) != 1 {
		t.Fatal(us, err)
	}
	n := 0
	for _, err := range s.Iter(ctx, httpx.ODataQuery{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Fatal(n)
	}
	s.Get(ctx, 1)
	check("web/getuserbyid(1)")
	s.ByLoginName(ctx, "i:0#.f|m|a@b.c")
	check("web/siteusers/getbyloginname('i:0#.f|m|a@b.c')")
	s.ByEmail(ctx, "a@b.c")
	if _, err := s.Ensure(ctx, "i:0#.f|m|a@b.c"); err != nil || srv.Last().Body != `{"logonName":"i:0#.f|m|a@b.c"}` {
		t.Fatal(err, srv.Last().Body)
	}
	s.Remove(ctx, 1)
	check("web/siteusers/removebyid(1)")
	s.Groups(ctx, 1)
	check("web/getuserbyid(1)/groups")
	s.AllGroups(ctx)
	s.GetGroup(ctx, 5)
	check("web/sitegroups(5)")
	s.GroupByName(ctx, "Members")
	check("web/sitegroups/getbyname('Members')")
	if g, err := s.CreateGroup(ctx, "G", "d"); err != nil || g.ID != 1 {
		t.Fatal(g, err)
	}
	s.GroupMembers(ctx, 5)
	check("web/sitegroups(5)/users")
	if _, err := s.AddToGroup(ctx, 5, "i:0#.f|m|a@b.c"); err != nil || srv.Last().Body != `{"LoginName":"i:0#.f|m|a@b.c"}` {
		t.Fatal(err)
	}
	if err := s.RemoveFromGroup(ctx, 5, 1); err != nil {
		t.Fatal(err)
	}
	check("web/sitegroups(5)/users/removebyid(1)")
	if err := s.DeleteGroup(ctx, 5); err != nil {
		t.Fatal(err)
	}
	check("web/sitegroups/removebyid(5)")
}
