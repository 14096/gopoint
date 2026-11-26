// Package recyclebin implements the Graph site recycle bin.
//
// This API is only available on the beta endpoint: set graph.Options.BaseURL
// to https://graph.microsoft.com/beta.
package recyclebin

import (
	"context"
	"iter"
	"time"

	"github.com/14096/gopoint/graph/internal/gpath"
	"github.com/14096/gopoint/httpx"
)

// Service exposes recycle bin operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Item is a deleted item.
type Item struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Size             int64     `json:"size"`
	DeletedDateTime  time.Time `json:"deletedDateTime"`
	OriginalLocation string    `json:"originalLocation"`
	CreatedDateTime  time.Time `json:"createdDateTime"`
}

func base(site string) string { return gpath.Site(site) + "/recycleBin/items" }

// Iter iterates over the items in the recycle bin of a site.
func (s *Service) Iter(ctx context.Context, site string) iter.Seq2[Item, error] {
	return httpx.Paginate[Item](ctx, s.c, httpx.NewRequest("GET", base(site)))
}

// All returns the items in the recycle bin of a site.
func (s *Service) All(ctx context.Context, site string) ([]Item, error) {
	return httpx.Collect[Item](ctx, s.c, httpx.NewRequest("GET", base(site)))
}

// Restore restores items to their original location.
func (s *Service) Restore(ctx context.Context, site string, ids ...string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("POST", base(site)+"/restore").WithJSON(map[string]any{"ids": ids}))
	return err
}

// Delete permanently deletes items.
func (s *Service) Delete(ctx context.Context, site string, ids ...string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("POST", base(site)+"/delete").WithJSON(map[string]any{"ids": ids}))
	return err
}
