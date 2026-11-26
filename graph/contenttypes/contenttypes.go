// Package contenttypes implements the Graph "content type" resource.
package contenttypes

import (
	"context"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes content type operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// ContentType holds the commonly used properties.
type ContentType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Group       string `json:"group"`
	Hidden      bool   `json:"hidden"`
}

func path(site, list string) string {
	p := "/sites/" + url.PathEscape(site)
	if list != "" {
		p += "/lists/" + url.PathEscape(list)
	}
	return p + "/contentTypes"
}

// List returns the content types of a list, or of the site when list is "".
func (s *Service) List(ctx context.Context, site, list string) ([]ContentType, error) {
	return httpx.Collect[ContentType](ctx, s.c, httpx.NewRequest("GET", path(site, list)))
}

// Get returns one content type.
func (s *Service) Get(ctx context.Context, site, list, id string) (*ContentType, error) {
	var ct ContentType
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", path(site, list)+"/"+url.PathEscape(id)), &ct); err != nil {
		return nil, err
	}
	return &ct, nil
}
