// Package lists implements the SharePoint "list" resource.
package lists

import (
	"context"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes list operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Common base templates.
const (
	GenericList     = 100
	DocumentLibrary = 101
	Tasks           = 107
)

// List holds the commonly used list properties.
type List struct {
	ID                  string `json:"Id"`
	Title               string `json:"Title"`
	Description         string `json:"Description"`
	BaseTemplate        int    `json:"BaseTemplate"`
	ItemCount           int    `json:"ItemCount"`
	Hidden              bool   `json:"Hidden"`
	EntityTypeName      string `json:"EntityTypeName"`
	ContentTypesEnabled bool   `json:"ContentTypesEnabled"`
}

// Get returns a list by reference.
func (s *Service) Get(ctx context.Context, ref spx.ListRef) (*List, error) {
	var l List
	if err := httpx.JSON(ctx, s.c, spx.Get(ref.Path()), &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// All returns every list of the web (including hidden ones).
func (s *Service) All(ctx context.Context, q httpx.ODataQuery) ([]List, error) {
	return httpx.Collect[List](ctx, s.c, spx.Get("web/lists").WithQuery(q.Values()))
}

// CreateParams describes a new list.
type CreateParams struct {
	Title        string
	Description  string
	BaseTemplate int // defaults to GenericList
}

// Create creates a list.
func (s *Service) Create(ctx context.Context, p CreateParams) (*List, error) {
	if p.BaseTemplate == 0 {
		p.BaseTemplate = GenericList
	}
	body := map[string]any{
		"Title": p.Title, "Description": p.Description,
		"BaseTemplate": p.BaseTemplate, "AllowContentTypes": true,
	}
	var l List
	if err := httpx.JSON(ctx, s.c, spx.Post("web/lists", body), &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Update merges the given properties (e.g. {"Title": "New"}) into the list.
func (s *Service) Update(ctx context.Context, ref spx.ListRef, props map[string]any) error {
	_, err := s.c.Do(ctx, spx.Merge(ref.Path(), props))
	return err
}

// Delete deletes the list.
func (s *Service) Delete(ctx context.Context, ref spx.ListRef) error {
	_, err := s.c.Do(ctx, spx.Remove(ref.Path()))
	return err
}
