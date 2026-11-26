// Package recyclebin implements the SharePoint recycle bin of a web or site collection.
package recyclebin

import (
	"context"
	"encoding/json"
	"iter"
	"time"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Bin selects which recycle bin to work on.
type Bin string

const (
	// Web is the recycle bin of the current web (first stage).
	Web Bin = "web/recyclebin"
	// Site is the site collection recycle bin (both stages).
	Site Bin = "site/recyclebin"
)

// Service exposes recycle bin operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Item is a deleted file, folder or list item.
type Item struct {
	ID            string      `json:"Id"`
	Title         string      `json:"Title"`
	LeafName      string      `json:"LeafName"`
	DirName       string      `json:"DirName"`
	ItemType      int         `json:"ItemType"`
	ItemState     int         `json:"ItemState"`
	Size          json.Number `json:"Size"`
	DeletedDate   time.Time   `json:"DeletedDate"`
	DeletedByName string      `json:"DeletedByName"`
	AuthorName    string      `json:"AuthorName"`
}

func item(b Bin, id string) string { return string(b) + "(guid" + spx.Quote(id) + ")" }

// Iter iterates over the items in a bin.
func (s *Service) Iter(ctx context.Context, b Bin, q httpx.ODataQuery) iter.Seq2[Item, error] {
	return httpx.Paginate[Item](ctx, s.c, spx.Get(string(b)).WithQuery(q.Values()))
}

// All returns the items in a bin.
func (s *Service) All(ctx context.Context, b Bin, q httpx.ODataQuery) ([]Item, error) {
	return httpx.Collect[Item](ctx, s.c, spx.Get(string(b)).WithQuery(q.Values()))
}

// Get returns one item.
func (s *Service) Get(ctx context.Context, b Bin, id string) (*Item, error) {
	var it Item
	if err := httpx.JSON(ctx, s.c, spx.Get(item(b, id)), &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// Restore restores an item to its original location.
func (s *Service) Restore(ctx context.Context, b Bin, id string) error {
	_, err := s.c.Do(ctx, spx.Post(item(b, id)+"/restore", nil))
	return err
}

// DeletePermanently deletes an item for good.
func (s *Service) DeletePermanently(ctx context.Context, b Bin, id string) error {
	_, err := s.c.Do(ctx, spx.Post(item(b, id)+"/deleteobject", nil))
	return err
}

// MoveToSecondStage moves an item from the web bin to the site collection bin.
func (s *Service) MoveToSecondStage(ctx context.Context, id string) error {
	_, err := s.c.Do(ctx, spx.Post(item(Web, id)+"/movetosecondstage", nil))
	return err
}

// RestoreAll restores every item in a bin.
func (s *Service) RestoreAll(ctx context.Context, b Bin) error {
	_, err := s.c.Do(ctx, spx.Post(string(b)+"/restoreall", nil))
	return err
}

// Empty permanently deletes every item in a bin.
func (s *Service) Empty(ctx context.Context, b Bin) error {
	_, err := s.c.Do(ctx, spx.Post(string(b)+"/deleteall", nil))
	return err
}
