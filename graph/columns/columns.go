// Package columns implements the Graph "column" resource (site or list columns).
package columns

import (
	"context"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes column operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Column holds the commonly used column properties.
type Column struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Hidden      bool   `json:"hidden"`
	ReadOnly    bool   `json:"readOnly"`
}

func path(site, list string) string {
	p := "/sites/" + url.PathEscape(site)
	if list != "" {
		p += "/lists/" + url.PathEscape(list)
	}
	return p + "/columns"
}

// List returns the columns of a list, or the site columns when list is "".
func (s *Service) List(ctx context.Context, site, list string) ([]Column, error) {
	return httpx.Collect[Column](ctx, s.c, httpx.NewRequest("GET", path(site, list)))
}

// Create adds a column. def is the Graph columnDefinition JSON, e.g.
// {"name":"Budget","currency":{"locale":"en-US"}}.
func (s *Service) Create(ctx context.Context, site, list string, def map[string]any) (*Column, error) {
	var c Column
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", path(site, list)).WithJSON(def), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Delete removes a column by ID.
func (s *Service) Delete(ctx context.Context, site, list, columnID string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", path(site, list)+"/"+url.PathEscape(columnID)))
	return err
}
