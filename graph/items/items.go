// Package items implements the Graph "list item" resource.
package items

import (
	"context"
	"iter"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes list item operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Item is a list item; Fields holds the column values.
type Item struct {
	ID              string         `json:"id"`
	WebURL          string         `json:"webUrl"`
	CreatedDateTime string         `json:"createdDateTime"`
	Fields          map[string]any `json:"fields"`
}

func base(site, list string) string {
	return "/sites/" + url.PathEscape(site) + "/lists/" + url.PathEscape(list) + "/items"
}

// withFields expands fields by default so items carry their column values.
func withFields(q httpx.ODataQuery) httpx.ODataQuery {
	if len(q.Expand) == 0 {
		q.Expand = []string{"fields"}
	}
	return q
}

// Iter iterates over items, following pages. Use ODataQuery for $select
// (e.g. "fields(select=Title)"), $filter and $expand.
func (s *Service) Iter(ctx context.Context, site, list string, q httpx.ODataQuery) iter.Seq2[Item, error] {
	return httpx.Paginate[Item](ctx, s.c, httpx.NewRequest("GET", base(site, list)).WithQuery(withFields(q).Values()))
}

// All collects every item matching q.
func (s *Service) All(ctx context.Context, site, list string, q httpx.ODataQuery) ([]Item, error) {
	return httpx.Collect[Item](ctx, s.c, httpx.NewRequest("GET", base(site, list)).WithQuery(withFields(q).Values()))
}

// Get returns one item.
func (s *Service) Get(ctx context.Context, site, list, id string, q httpx.ODataQuery) (*Item, error) {
	var it Item
	r := httpx.NewRequest("GET", base(site, list)+"/"+url.PathEscape(id)).WithQuery(withFields(q).Values())
	if err := httpx.JSON(ctx, s.c, r, &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// Create adds an item with the given field values.
func (s *Service) Create(ctx context.Context, site, list string, fields map[string]any) (*Item, error) {
	var it Item
	r := httpx.NewRequest("POST", base(site, list)).WithJSON(map[string]any{"fields": fields})
	if err := httpx.JSON(ctx, s.c, r, &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// Update patches the given field values and returns the resulting fields.
func (s *Service) Update(ctx context.Context, site, list, id string, fields map[string]any) (map[string]any, error) {
	var out map[string]any
	r := httpx.NewRequest("PATCH", base(site, list)+"/"+url.PathEscape(id)+"/fields").WithJSON(fields)
	if err := httpx.JSON(ctx, s.c, r, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Delete removes the item.
func (s *Service) Delete(ctx context.Context, site, list, id string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", base(site, list)+"/"+url.PathEscape(id)))
	return err
}

// Delta returns the list items changed since deltaLink and the new link to
// store. With an empty deltaLink it enumerates the whole list and returns the
// first link.
func (s *Service) Delta(ctx context.Context, site, list, deltaLink string) ([]Item, string, error) {
	path := deltaLink
	if path == "" {
		path = base(site, list) + "/delta"
	}
	return httpx.Delta[Item](ctx, s.c, httpx.NewRequest("GET", path))
}
