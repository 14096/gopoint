// Package pages implements Graph modern site pages (sitePage).
package pages

import (
	"context"
	"iter"
	"net/url"
	"time"

	"github.com/14096/gopoint/graph/internal/gpath"
	"github.com/14096/gopoint/httpx"
)

// Service exposes site page operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Page holds the commonly used page properties.
type Page struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	WebURL               string    `json:"webUrl"`
	PageLayout           string    `json:"pageLayout"`
	LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
	PublishingState      *struct {
		Level string `json:"level"`
	} `json:"publishingState"`
}

// CreateParams describes a new page. Name is the file name (e.g. "news.aspx").
type CreateParams struct {
	Name  string
	Title string
	// Layout is "article" (default), "home", ...
	Layout string
}

const typeSegment = "/microsoft.graph.sitePage"

func base(site string) string { return gpath.Site(site) + "/pages" }
func page(site, id string) string {
	return base(site) + "/" + url.PathEscape(id) + typeSegment
}

// Iter iterates over the pages of a site.
func (s *Service) Iter(ctx context.Context, site string) iter.Seq2[Page, error] {
	return httpx.Paginate[Page](ctx, s.c, httpx.NewRequest("GET", base(site)+typeSegment))
}

// All returns the pages of a site.
func (s *Service) All(ctx context.Context, site string) ([]Page, error) {
	return httpx.Collect[Page](ctx, s.c, httpx.NewRequest("GET", base(site)+typeSegment))
}

// Get returns a page.
func (s *Service) Get(ctx context.Context, site, id string) (*Page, error) {
	var p Page
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", page(site, id)), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Create creates a draft page.
func (s *Service) Create(ctx context.Context, site string, p CreateParams) (*Page, error) {
	if p.Layout == "" {
		p.Layout = "article"
	}
	body := map[string]any{"@odata.type": "#microsoft.graph.sitePage", "name": p.Name, "title": p.Title, "pageLayout": p.Layout}
	var out Page
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", base(site)).WithJSON(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update changes the title of a page.
func (s *Service) Update(ctx context.Context, site, id, title string) (*Page, error) {
	body := map[string]any{"@odata.type": "#microsoft.graph.sitePage", "title": title}
	var out Page
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("PATCH", page(site, id)).WithJSON(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Publish publishes the page.
func (s *Service) Publish(ctx context.Context, site, id string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("POST", page(site, id)+"/publish"))
	return err
}

// Delete deletes the page.
func (s *Service) Delete(ctx context.Context, site, id string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", base(site)+"/"+url.PathEscape(id)))
	return err
}
