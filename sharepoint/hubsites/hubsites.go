// Package hubsites implements SharePoint hub sites. Registering and
// unregistering requires SharePoint administrator rights; joining requires
// permission on the hub.
package hubsites

import (
	"context"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes hub site operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// NoHub is the ID that means "not associated with a hub".
const NoHub = "00000000-0000-0000-0000-000000000000"

// HubSite describes a hub site.
type HubSite struct {
	ID          string `json:"ID"`
	SiteID      string `json:"SiteId"`
	SiteURL     string `json:"SiteUrl"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	LogoURL     string `json:"LogoUrl"`
	Targets     string `json:"Targets"`
}

// UpdateParams holds the changeable fields; nil values are left untouched.
type UpdateParams struct {
	Title       *string
	Description *string
	LogoURL     *string
}

// All lists the hub sites of the tenant.
func (s *Service) All(ctx context.Context) ([]HubSite, error) {
	return httpx.Collect[HubSite](ctx, s.c, spx.Get("hubsites"))
}

// Get returns a hub site by its hub ID.
func (s *Service) Get(ctx context.Context, hubID string) (*HubSite, error) {
	var h HubSite
	if err := httpx.JSON(ctx, s.c, spx.Get("hubsites(guid"+spx.Quote(hubID)+")"), &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// Current returns the ID of the hub the client's site is associated with, or
// NoHub when the site is not part of a hub.
func (s *Service) Current(ctx context.Context) (string, error) {
	var out struct {
		HubSiteID string `json:"HubSiteId"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Get("site").SetQuery("$select", "HubSiteId"), &out); err != nil {
		return "", err
	}
	return out.HubSiteID, nil
}

// Register turns the client's site into a hub site.
func (s *Service) Register(ctx context.Context) (*HubSite, error) {
	var h HubSite
	if err := httpx.JSON(ctx, s.c, spx.Post("site/registerhubsite", nil), &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// Unregister stops the client's site from being a hub site.
func (s *Service) Unregister(ctx context.Context) error {
	_, err := s.c.Do(ctx, spx.Post("site/unregisterhubsite", nil))
	return err
}

// Update changes the hub's title, description or logo; the client must be the hub site.
func (s *Service) Update(ctx context.Context, hubID string, p UpdateParams) error {
	body := map[string]any{}
	if p.Title != nil {
		body["Title"] = *p.Title
	}
	if p.Description != nil {
		body["Description"] = *p.Description
	}
	if p.LogoURL != nil {
		body["LogoUrl"] = *p.LogoURL
	}
	_, err := s.c.Do(ctx, spx.Merge("hubsites(guid"+spx.Quote(hubID)+")", body))
	return err
}

// Join associates the client's site with a hub.
func (s *Service) Join(ctx context.Context, hubID string) error {
	_, err := s.c.Do(ctx, spx.Post("site/joinhubsite("+spx.Quote(hubID)+")", nil))
	return err
}

// Leave removes the client's site from its hub.
func (s *Service) Leave(ctx context.Context) error { return s.Join(ctx, NoHub) }
