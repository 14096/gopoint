// Package pages implements SharePoint modern site pages (the SitePages API).
package pages

import (
	"context"
	"iter"
	"strconv"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes site page operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Page holds the commonly used page properties.
type Page struct {
	ID             int    `json:"Id"`
	Title          string `json:"Title"`
	Description    string `json:"Description"`
	FileName       string `json:"FileName"`
	Name           string `json:"Name"`
	URL            string `json:"Url"`
	AbsoluteURL    string `json:"AbsoluteUrl"`
	PageLayoutType string `json:"PageLayoutType"`
	PromotedState  int    `json:"PromotedState"`
	VersionInfo    string `json:"Version"`
	CanvasContent1 string `json:"CanvasContent1,omitempty"`
}

// CreateParams describes a new page. Name is the file name (e.g. "news.aspx").
type CreateParams struct {
	Title string
	Name  string
	// Layout is "Article" (default), "Home", "SingleWebPartApp", "RepostPage", ...
	Layout      string
	Description string
}

// UpdateParams holds the changeable fields; empty values are left untouched.
type UpdateParams struct {
	Title          string
	Description    string
	CanvasContent1 string
}

const base = "sitepages/pages"

func pageID(id int) string { return base + "(" + strconv.Itoa(id) + ")" }

// All lists the site pages.
func (s *Service) All(ctx context.Context, q httpx.ODataQuery) ([]Page, error) {
	return httpx.Collect[Page](ctx, s.c, spx.Get(base).WithQuery(q.Values()))
}

// Iter iterates over the site pages.
func (s *Service) Iter(ctx context.Context, q httpx.ODataQuery) iter.Seq2[Page, error] {
	return httpx.Paginate[Page](ctx, s.c, spx.Get(base).WithQuery(q.Values()))
}

// Get returns a page by item ID.
func (s *Service) Get(ctx context.Context, id int) (*Page, error) {
	var p Page
	if err := httpx.JSON(ctx, s.c, spx.Get(pageID(id)), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByURL returns a page by site-relative URL (e.g. "SitePages/Home.aspx").
func (s *Service) GetByURL(ctx context.Context, pageURL string) (*Page, error) {
	var p Page
	if err := httpx.JSON(ctx, s.c, spx.Get(base+"/GetByUrl("+spx.QuotePath(pageURL)+")"), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Create creates a draft page.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Page, error) {
	if p.Layout == "" {
		p.Layout = "Article"
	}
	body := map[string]any{"Title": p.Title, "PageLayoutType": p.Layout}
	if p.Name != "" {
		body["Name"] = p.Name
	}
	if p.Description != "" {
		body["Description"] = p.Description
	}
	var out Page
	if err := httpx.JSON(ctx, s.c, spx.Post(base, body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update saves changes to a page as a draft.
func (s *Service) Update(ctx context.Context, id int, p UpdateParams) (*Page, error) {
	body := map[string]any{}
	if p.Title != "" {
		body["Title"] = p.Title
	}
	if p.Description != "" {
		body["Description"] = p.Description
	}
	if p.CanvasContent1 != "" {
		body["CanvasContent1"] = p.CanvasContent1
	}
	var out Page
	if err := httpx.JSON(ctx, s.c, spx.Post(pageID(id)+"/savepagedraft", body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Publish publishes the page.
func (s *Service) Publish(ctx context.Context, id int) error {
	_, err := s.c.Do(ctx, spx.Post(pageID(id)+"/publish", nil))
	return err
}

// Delete deletes the page.
func (s *Service) Delete(ctx context.Context, id int) error {
	_, err := s.c.Do(ctx, spx.Remove(pageID(id)))
	return err
}
