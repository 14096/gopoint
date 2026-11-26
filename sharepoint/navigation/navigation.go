// Package navigation implements the SharePoint quick launch and top navigation bars.
package navigation

import (
	"context"
	"strconv"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Bar selects a navigation bar.
type Bar string

const (
	// QuickLaunch is the left-hand navigation.
	QuickLaunch Bar = "quicklaunch"
	// TopBar is the top link bar.
	TopBar Bar = "topnavigationbar"
)

// Service exposes navigation operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Node is a navigation link.
type Node struct {
	ID         int    `json:"Id"`
	Title      string `json:"Title"`
	URL        string `json:"Url"`
	IsExternal bool   `json:"IsExternal"`
	IsVisible  bool   `json:"IsVisible"`
}

// NodeParams describes a node to create or update. Empty fields are left untouched on update.
type NodeParams struct {
	Title      string
	URL        string
	IsExternal bool
}

func (p NodeParams) body() map[string]any {
	b := map[string]any{"IsExternal": p.IsExternal}
	if p.Title != "" {
		b["Title"] = p.Title
	}
	if p.URL != "" {
		b["Url"] = p.URL
	}
	return b
}

func node(id int) string { return "web/navigation/GetNodeById(" + strconv.Itoa(id) + ")" }

// All returns the top-level nodes of a bar.
func (s *Service) All(ctx context.Context, bar Bar) ([]Node, error) {
	return httpx.Collect[Node](ctx, s.c, spx.Get("web/navigation/"+string(bar)))
}

// Get returns a node.
func (s *Service) Get(ctx context.Context, id int) (*Node, error) {
	var n Node
	if err := httpx.JSON(ctx, s.c, spx.Get(node(id)), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// Children lists the child nodes of a node.
func (s *Service) Children(ctx context.Context, id int) ([]Node, error) {
	return httpx.Collect[Node](ctx, s.c, spx.Get(node(id)+"/children"))
}

// Add appends a top-level node to a bar.
func (s *Service) Add(ctx context.Context, bar Bar, p NodeParams) (*Node, error) {
	var n Node
	if err := httpx.JSON(ctx, s.c, spx.Post("web/navigation/"+string(bar), p.body()), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// AddChild appends a child under an existing node.
func (s *Service) AddChild(ctx context.Context, parentID int, p NodeParams) (*Node, error) {
	var n Node
	if err := httpx.JSON(ctx, s.c, spx.Post(node(parentID)+"/children", p.body()), &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// Update changes a node.
func (s *Service) Update(ctx context.Context, id int, p NodeParams) error {
	_, err := s.c.Do(ctx, spx.Merge(node(id), p.body()))
	return err
}

// Delete removes a node.
func (s *Service) Delete(ctx context.Context, id int) error {
	_, err := s.c.Do(ctx, spx.Remove(node(id)))
	return err
}
