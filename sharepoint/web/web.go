// Package web implements the SharePoint "web" resource (site properties and subsites).
package web

import (
	"context"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes web operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Web holds the commonly used web properties.
type Web struct {
	ID                string `json:"Id"`
	Title             string `json:"Title"`
	Description       string `json:"Description"`
	URL               string `json:"Url"`
	ServerRelativeURL string `json:"ServerRelativeUrl"`
	WebTemplate       string `json:"WebTemplate"`
	Created           string `json:"Created"`
}

// Get returns the current web.
func (s *Service) Get(ctx context.Context) (*Web, error) {
	var w Web
	if err := httpx.JSON(ctx, s.c, spx.Get("web"), &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// Subwebs lists the direct subsites of the current web.
func (s *Service) Subwebs(ctx context.Context) ([]Web, error) {
	return httpx.Collect[Web](ctx, s.c, spx.Get("web/webs"))
}

// CreateParams describes a new subsite.
type CreateParams struct {
	Title       string
	URL         string // relative URL segment of the new subsite
	Description string
	// Template is e.g. "STS#3" (team site, no group).
	Template                       string
	Language                       int // LCID, default 1033
	UseSamePermissionsAsParentSite bool
}

// CreateSubweb creates a subsite under the current web.
func (s *Service) CreateSubweb(ctx context.Context, p CreateParams) (*Web, error) {
	if p.Language == 0 {
		p.Language = 1033
	}
	if p.Template == "" {
		p.Template = "STS#3"
	}
	body := map[string]any{"parameters": map[string]any{
		"Title": p.Title, "Url": p.URL, "Description": p.Description,
		"WebTemplate": p.Template, "Language": p.Language,
		"UseSamePermissionsAsParentSite": p.UseSamePermissionsAsParentSite,
	}}
	var w Web
	if err := httpx.JSON(ctx, s.c, spx.Post("web/webinfos/add", body), &w); err != nil {
		return nil, err
	}
	return &w, nil
}
