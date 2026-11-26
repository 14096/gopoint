package permissions_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/14096/gopoint/internal/testutil"
	"github.com/14096/gopoint/sharepoint/permissions"
	"github.com/14096/gopoint/sharepoint/spx"
)

func TestPermissions(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) {
		w.Write([]byte(`{"value":[{"PrincipalId":7,"Member":{"Title":"Team"},"RoleDefinitionBindings":[{"Id":1073741827,"Name":"Contribute"}]}],"Name":"Edit","High":"5","Low":"6"}`))
	})
	s := permissions.New(srv.Client("/_api"))
	ctx := context.Background()
	list := spx.ByTitle("Docs")
	item := spx.ItemRef(list, 4)

	as, err := s.Assignments(ctx, item)
	if err != nil || as[0].Member.Title != "Team" || as[0].Roles[0].Name != "Contribute" ||
		srv.Last().Path != "/_api/web/lists/getbytitle('Docs')/items(4)/roleassignments" || srv.Last().Query.Get("$expand") != "Member,RoleDefinitionBindings" {
		t.Fatal(as, err, srv.Last().Path)
	}
	if err := s.Grant(ctx, list, 7, 9); err != nil || srv.Last().Path != "/_api/web/lists/getbytitle('Docs')/roleassignments/addroleassignment(principalid=7,roledefid=9)" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.GrantByName(ctx, spx.Web, 7, "Edit"); err != nil || srv.Last().Path != "/_api/web/roleassignments/addroleassignment(principalid=7,roledefid=0)" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.Revoke(ctx, spx.Web, 7, 9); err != nil || srv.Last().Path != "/_api/web/roleassignments/removeroleassignment(principalid=7,roledefid=9)" {
		t.Fatal(err)
	}
	if err := s.RemovePrincipal(ctx, spx.Web, 7); err != nil || srv.Last().Header.Get("X-HTTP-Method") != "DELETE" {
		t.Fatal(err)
	}
	if err := s.BreakInheritance(ctx, spx.FolderItem("/sites/t/D"), true, false); err != nil ||
		srv.Last().Path != "/_api/web/GetFolderByServerRelativeUrl('/sites/t/D')/ListItemAllFields/breakroleinheritance(copyroleassignments=true,clearsubscopes=false)" {
		t.Fatal(err, srv.Last().Path)
	}
	if err := s.ResetInheritance(ctx, spx.FileItem("/sites/t/D/a.txt")); err != nil {
		t.Fatal(err)
	}
	if p, err := s.EffectivePermissions(ctx, spx.Web, "i:0#.f|m|a@b.c"); err != nil || p.Low.String() != "6" || srv.Last().Query.Get("@u") != "'i:0#.f|m|a@b.c'" {
		t.Fatal(p, err)
	}
	if r, err := s.RoleDefinitionByName(ctx, "Edit"); err != nil || r.Name != "Edit" {
		t.Fatal(r, err)
	}
	if rs, err := s.RoleDefinitions(ctx); err != nil || len(rs) != 1 {
		t.Fatal(rs, err)
	}
}

func TestHasUnique(t *testing.T) {
	srv := testutil.New(t, func(c testutil.Call, w http.ResponseWriter) { w.Write([]byte(`{"value":true}`)) })
	ok, err := permissions.New(srv.Client("/_api")).HasUniquePermissions(context.Background(), spx.Web)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
}
