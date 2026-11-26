// Package lists implements the Graph "list" resource.
package lists

import (
	"context"
	"iter"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes list operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// List holds the commonly used list properties.
type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl"`
	Info        struct {
		Template string `json:"template"`
		Hidden   bool   `json:"hidden"`
	} `json:"list"`
}

func base(site string) string { return "/sites/" + url.PathEscape(site) + "/lists" }

// Iter iterates over the lists of a site.
func (s *Service) Iter(ctx context.Context, site string, q httpx.ODataQuery) iter.Seq2[List, error] {
	return httpx.Paginate[List](ctx, s.c, httpx.NewRequest("GET", base(site)).WithQuery(q.Values()))
}

// All returns every list of a site.
func (s *Service) All(ctx context.Context, site string, q httpx.ODataQuery) ([]List, error) {
	return httpx.Collect[List](ctx, s.c, httpx.NewRequest("GET", base(site)).WithQuery(q.Values()))
}

// Get returns a list by ID or display name.
func (s *Service) Get(ctx context.Context, site, list string) (*List, error) {
	var l List
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", base(site)+"/"+url.PathEscape(list)), &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// CreateParams describes a new list.
type CreateParams struct {
	DisplayName string
	// Template is e.g. "genericList" or "documentLibrary".
	Template string
	// Columns are column definitions in Graph JSON form.
	Columns []map[string]any
}

// Create creates a list.
func (s *Service) Create(ctx context.Context, site string, p CreateParams) (*List, error) {
	if p.Template == "" {
		p.Template = "genericList"
	}
	body := map[string]any{"displayName": p.DisplayName, "list": map[string]any{"template": p.Template}}
	if len(p.Columns) > 0 {
		body["columns"] = p.Columns
	}
	var l List
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", base(site)).WithJSON(body), &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Delete deletes a list.
func (s *Service) Delete(ctx context.Context, site, list string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", base(site)+"/"+url.PathEscape(list)))
	return err
}
