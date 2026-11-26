// Package permissions implements Graph permissions on drive items and sites.
// To grant access to people or create links use the sharing package.
package permissions

import (
	"context"
	"iter"
	"net/url"

	"github.com/14096/gopoint/graph/internal/gpath"
	"github.com/14096/gopoint/httpx"
)

// Service exposes permission operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Role is the access level of a permission.
type Role string

// Roles used by drive items and sites.
const (
	// Read allows viewing.
	Read Role = "read"
	// Write allows viewing and editing.
	Write Role = "write"
	// Owner allows everything, including managing permissions.
	Owner Role = "owner"
	// FullControl is the site-level full control role.
	FullControl Role = "fullcontrol"
)

// Identity is a user, group or application.
type Identity struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// IdentitySet groups the identity kinds Graph returns.
type IdentitySet struct {
	User        *Identity `json:"user"`
	Group       *Identity `json:"group"`
	Application *Identity `json:"application"`
	SiteUser    *Identity `json:"siteUser"`
	SiteGroup   *Identity `json:"siteGroup"`
}

// Permission is an access grant or sharing link on an item.
type Permission struct {
	ID                  string        `json:"id"`
	Roles               []Role        `json:"roles"`
	GrantedTo           *IdentitySet  `json:"grantedToV2"`
	GrantedToIdentities []IdentitySet `json:"grantedToIdentitiesV2"`
	Link                *struct {
		Type   string `json:"type"`
		Scope  string `json:"scope"`
		WebURL string `json:"webUrl"`
	} `json:"link"`
	InheritedFrom *struct {
		ID string `json:"id"`
	} `json:"inheritedFrom"`
}

func item(drive, id string) string { return gpath.Item(drive, id) + "/permissions" }

// All returns the permissions of a drive item.
func (s *Service) All(ctx context.Context, drive, itemID string) ([]Permission, error) {
	return httpx.Collect[Permission](ctx, s.c, httpx.NewRequest("GET", item(drive, itemID)))
}

// Iter iterates over the permissions of a drive item.
func (s *Service) Iter(ctx context.Context, drive, itemID string) iter.Seq2[Permission, error] {
	return httpx.Paginate[Permission](ctx, s.c, httpx.NewRequest("GET", item(drive, itemID)))
}

// Get returns one permission of a drive item.
func (s *Service) Get(ctx context.Context, drive, itemID, permID string) (*Permission, error) {
	return one(ctx, s.c, httpx.NewRequest("GET", item(drive, itemID)+"/"+url.PathEscape(permID)))
}

// Update replaces the roles of a permission.
func (s *Service) Update(ctx context.Context, drive, itemID, permID string, roles ...Role) (*Permission, error) {
	req := httpx.NewRequest("PATCH", item(drive, itemID)+"/"+url.PathEscape(permID)).WithJSON(map[string]any{"roles": roles})
	return one(ctx, s.c, req)
}

// Delete removes a permission or sharing link.
func (s *Service) Delete(ctx context.Context, drive, itemID, permID string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", item(drive, itemID)+"/"+url.PathEscape(permID)))
	return err
}

// AllSite returns the application permissions of a site (Sites.Selected).
func (s *Service) AllSite(ctx context.Context, site string) ([]Permission, error) {
	return httpx.Collect[Permission](ctx, s.c, httpx.NewRequest("GET", gpath.Site(site)+"/permissions"))
}

// GrantSiteApp grants an application roles on a site.
func (s *Service) GrantSiteApp(ctx context.Context, site, appID, appName string, roles ...Role) (*Permission, error) {
	body := map[string]any{"roles": roles, "grantedToIdentities": []any{
		map[string]any{"application": map[string]any{"id": appID, "displayName": appName}},
	}}
	return one(ctx, s.c, httpx.NewRequest("POST", gpath.Site(site)+"/permissions").WithJSON(body))
}

// DeleteSite removes an application permission from a site.
func (s *Service) DeleteSite(ctx context.Context, site, permID string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", gpath.Site(site)+"/permissions/"+url.PathEscape(permID)))
	return err
}

func one(ctx context.Context, d httpx.Doer, req *httpx.Request) (*Permission, error) {
	var p Permission
	if err := httpx.JSON(ctx, d, req, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
