// Package sharing implements Graph sharing of drive items: links and invitations.
package sharing

import (
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/14096/gopoint/graph/drives"
	"github.com/14096/gopoint/graph/internal/gpath"
	"github.com/14096/gopoint/graph/permissions"
	"github.com/14096/gopoint/httpx"
)

// Service exposes sharing operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// LinkType is the kind of access a sharing link gives.
type LinkType string

// Link types.
const (
	// ViewLink grants read-only access.
	ViewLink LinkType = "view"
	// EditLink grants read-write access.
	EditLink LinkType = "edit"
	// EmbedLink is a read-only link for embedding (OneDrive personal only).
	EmbedLink LinkType = "embed"
)

// LinkScope is who can use a sharing link.
type LinkScope string

// Link scopes.
const (
	// AnyoneScope works without signing in.
	AnyoneScope LinkScope = "anonymous"
	// OrganizationScope works for people in the tenant.
	OrganizationScope LinkScope = "organization"
	// UsersScope works for the people it was explicitly shared with.
	UsersScope LinkScope = "users"
)

// LinkParams tunes link creation.
type LinkParams struct {
	// Type defaults to ViewLink.
	Type LinkType
	// Scope defaults to the tenant's default link scope.
	Scope LinkScope
	// Expires is the optional link expiry.
	Expires  time.Time
	Password string
}

// CreateLink creates (or returns the existing) sharing link; the returned
// permission's Link.WebURL is the URL to hand out.
func (s *Service) CreateLink(ctx context.Context, drive, itemID string, p LinkParams) (*permissions.Permission, error) {
	if p.Type == "" {
		p.Type = ViewLink
	}
	body := map[string]any{"type": p.Type}
	if p.Scope != "" {
		body["scope"] = p.Scope
	}
	if !p.Expires.IsZero() {
		body["expirationDateTime"] = p.Expires.UTC().Format(time.RFC3339)
	}
	if p.Password != "" {
		body["password"] = p.Password
	}
	var out permissions.Permission
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", gpath.Item(drive, itemID)+"/createLink").WithJSON(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// InviteParams describes an invitation.
type InviteParams struct {
	Emails []string
	// Roles default to permissions.Read.
	Roles          []permissions.Role
	SendInvitation bool
	Message        string
	// RequireSignIn restricts access to signed-in recipients.
	RequireSignIn bool
}

// Invite shares an item with specific people.
func (s *Service) Invite(ctx context.Context, drive, itemID string, p InviteParams) ([]permissions.Permission, error) {
	if len(p.Emails) == 0 {
		return nil, errors.New("sharing: at least one recipient is required")
	}
	if len(p.Roles) == 0 {
		p.Roles = []permissions.Role{permissions.Read}
	}
	var rcpt []map[string]string
	for _, e := range p.Emails {
		rcpt = append(rcpt, map[string]string{"email": e})
	}
	body := map[string]any{"recipients": rcpt, "roles": p.Roles, "requireSignIn": p.RequireSignIn, "sendInvitation": p.SendInvitation}
	if p.Message != "" {
		body["message"] = p.Message
	}
	var out struct {
		Value []permissions.Permission `json:"value"`
	}
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", gpath.Item(drive, itemID)+"/invite").WithJSON(body), &out); err != nil {
		return nil, err
	}
	return out.Value, nil
}

// Unshare removes a sharing link or grant by permission ID.
func (s *Service) Unshare(ctx context.Context, drive, itemID, permID string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", gpath.Item(drive, itemID)+"/permissions/"+url.PathEscape(permID)))
	return err
}

// Resolve returns the drive item behind a sharing URL.
func (s *Service) Resolve(ctx context.Context, sharingURL string) (*drives.Item, error) {
	id := "u!" + strings.TrimRight(strings.NewReplacer("/", "_", "+", "-").Replace(base64.StdEncoding.EncodeToString([]byte(sharingURL))), "=")
	var it drives.Item
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", "/shares/"+id+"/driveItem"), &it); err != nil {
		return nil, err
	}
	return &it, nil
}
