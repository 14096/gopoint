// Package sites implements the Graph "site" resource.
package sites

import (
	"context"
	"iter"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes site operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Site holds the commonly used site properties.
type Site struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WebURL      string `json:"webUrl"`
}

func (s *Service) get(ctx context.Context, path string) (*Site, error) {
	var site Site
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", path), &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// Root returns the tenant root site.
func (s *Service) Root(ctx context.Context) (*Site, error) { return s.get(ctx, "/sites/root") }

// Get returns a site by ID ("host,siteCollectionId,webId").
func (s *Service) Get(ctx context.Context, id string) (*Site, error) {
	return s.get(ctx, "/sites/"+url.PathEscape(id))
}

// ByPath resolves a site from hostname and server-relative path, e.g.
// ("contoso.sharepoint.com", "/sites/team").
func (s *Service) ByPath(ctx context.Context, hostname, path string) (*Site, error) {
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	return s.get(ctx, "/sites/"+hostname+":"+path)
}

// Search iterates over sites matching the query.
func (s *Service) Search(ctx context.Context, query string) iter.Seq2[Site, error] {
	return httpx.Paginate[Site](ctx, s.c, httpx.NewRequest("GET", "/sites").SetQuery("search", query))
}

// Subsites lists the subsites of a site.
func (s *Service) Subsites(ctx context.Context, id string) ([]Site, error) {
	return httpx.Collect[Site](ctx, s.c, httpx.NewRequest("GET", "/sites/"+url.PathEscape(id)+"/sites"))
}
