// Package items implements the SharePoint "list item" resource.
package items

import (
	"context"
	"fmt"
	"iter"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes list item operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Item is a list item as a property bag.
type Item map[string]any

// ID returns the numeric item ID, or 0.
func (i Item) ID() int {
	switch v := i["Id"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func itemPath(ref spx.ListRef, id int) string {
	return fmt.Sprintf("%s/items(%d)", ref.Path(), id)
}

// Get returns one item.
func (s *Service) Get(ctx context.Context, ref spx.ListRef, id int, q httpx.ODataQuery) (Item, error) {
	var it Item
	if err := httpx.JSON(ctx, s.c, spx.Get(itemPath(ref, id)).WithQuery(q.Values()), &it); err != nil {
		return nil, err
	}
	return it, nil
}

// Query iterates over the items matching q, following pages.
func (s *Service) Query(ctx context.Context, ref spx.ListRef, q httpx.ODataQuery) iter.Seq2[Item, error] {
	return httpx.Paginate[Item](ctx, s.c, spx.Get(ref.Path()+"/items").WithQuery(q.Values()))
}

// All collects every item matching q.
func (s *Service) All(ctx context.Context, ref spx.ListRef, q httpx.ODataQuery) ([]Item, error) {
	return httpx.Collect[Item](ctx, s.c, spx.Get(ref.Path()+"/items").WithQuery(q.Values()))
}

// Create adds an item with the given field values.
func (s *Service) Create(ctx context.Context, ref spx.ListRef, fields map[string]any) (Item, error) {
	var it Item
	if err := httpx.JSON(ctx, s.c, spx.Post(ref.Path()+"/items", fields), &it); err != nil {
		return nil, err
	}
	return it, nil
}

// Update merges the given field values into the item; other fields are untouched.
func (s *Service) Update(ctx context.Context, ref spx.ListRef, id int, fields map[string]any) error {
	_, err := s.c.Do(ctx, spx.Merge(itemPath(ref, id), fields))
	return err
}

// Delete removes the item.
func (s *Service) Delete(ctx context.Context, ref spx.ListRef, id int) error {
	_, err := s.c.Do(ctx, spx.Remove(itemPath(ref, id)))
	return err
}
