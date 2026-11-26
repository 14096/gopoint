package graph_test

import (
	"context"

	"github.com/14096/gopoint/auth"
	"github.com/14096/gopoint/graph"
	"github.com/14096/gopoint/httpx"
)

type permissions struct{ c httpx.Doer }

type permission struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
}

func (s *permissions) ForSite(ctx context.Context, site string) ([]permission, error) {
	return httpx.Collect[permission](ctx, s.c, httpx.NewRequest("GET", "/sites/"+site+"/permissions"))
}

func Example_newResource() {
	var cred auth.Credential
	c, _ := graph.New(cred, graph.Options{})

	_, _ = c.Sites().Root(context.Background())

	p := &permissions{c: c.Raw()}
	_, _ = p.ForSite(context.Background(), "root")

	var me map[string]any
	_ = c.Raw().Get(context.Background(), "/me", &me)
}
