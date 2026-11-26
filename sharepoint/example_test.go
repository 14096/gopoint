package sharepoint_test

import (
	"context"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint"
	"github.com/14096/gopoint/sharepoint/spx"
)

type recycleBin struct{ c httpx.Doer }

type binItem struct {
	ID    string `json:"Id"`
	Title string `json:"Title"`
}

func (s *recycleBin) List(ctx context.Context) ([]binItem, error) {
	return httpx.Collect[binItem](ctx, s.c, spx.Get("web/recyclebin"))
}

func Example_newResource() {
	var cred auth.Credential
	c, _ := sharepoint.New("https://contoso.sharepoint.com/sites/team", cred, sharepoint.Options{})

	_, _ = c.Lists().All(context.Background(), httpx.ODataQuery{})

	bin := &recycleBin{c: c.Raw()}
	_, _ = bin.List(context.Background())

	var out map[string]any
	_ = c.Raw().Get(context.Background(), "web/currentuser", &out)
}
