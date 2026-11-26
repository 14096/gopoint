// Package sharing implements SharePoint sharing: sharing links and invitations
// for files, folders and list items.
package sharing

import (
	"context"
	"errors"
	"time"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes sharing operations. Targets are spx.FileItem, spx.FolderItem or spx.ItemRef scopes.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// LinkKind is the type of a sharing link.
type LinkKind int

const (
	// DirectLink is a link that only works for people who already have access.
	DirectLink LinkKind = 1
	// OrganizationView lets people in the organization view.
	OrganizationView LinkKind = 2
	// OrganizationEdit lets people in the organization edit.
	OrganizationEdit LinkKind = 3
	// AnonymousView lets anyone with the link view.
	AnonymousView LinkKind = 4
	// AnonymousEdit lets anyone with the link edit.
	AnonymousEdit LinkKind = 5
	// FlexibleLink is a link whose audience and rights are set afterwards.
	FlexibleLink LinkKind = 6
)

// Role is the access granted by an invitation.
type Role int

const (
	// View grants read-only access.
	View Role = 1
	// Edit grants read-write access.
	Edit Role = 2
)

// Link is a created sharing link.
type Link struct {
	URL         string
	Kind        LinkKind
	ShareID     string
	IsAnonymous bool
}

// LinkParams tunes link creation.
type LinkParams struct {
	Kind LinkKind
	// Expires is the optional link expiry; only valid for anonymous links.
	Expires time.Time
}

// CreateLink creates (or returns the existing) sharing link of the given kind.
func (s *Service) CreateLink(ctx context.Context, target spx.Scope, p LinkParams) (*Link, error) {
	if p.Kind == 0 {
		return nil, errors.New("sharing: link kind is required")
	}
	settings := map[string]any{"linkKind": int(p.Kind)}
	if !p.Expires.IsZero() {
		settings["expiration"] = p.Expires.UTC().Format(time.RFC3339)
	}
	body := map[string]any{"request": map[string]any{"createLink": true, "settings": settings}}
	var out struct {
		Info struct {
			URL         string   `json:"Url"`
			LinkKind    LinkKind `json:"LinkKind"`
			ShareID     string   `json:"ShareId"`
			IsAnonymous bool     `json:"IsAnonymous"`
		} `json:"sharingLinkInfo"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Post(target.Path()+"/ShareLink", body), &out); err != nil {
		return nil, err
	}
	i := out.Info
	return &Link{URL: i.URL, Kind: i.LinkKind, ShareID: i.ShareID, IsAnonymous: i.IsAnonymous}, nil
}

// DeleteLink removes the sharing link of the given kind.
func (s *Service) DeleteLink(ctx context.Context, target spx.Scope, kind LinkKind) error {
	body := map[string]any{"linkKind": int(kind), "shareId": "00000000-0000-0000-0000-000000000000"}
	_, err := s.c.Do(ctx, spx.Post(target.Path()+"/UnshareLink", body))
	return err
}

// Recipient is a person invited to a resource.
type Recipient struct {
	Email string
	Role  Role
}

// InviteParams describes an invitation.
type InviteParams struct {
	// ResourceURL is the absolute URL of the file, folder or page.
	ResourceURL string
	Recipients  []Recipient
	// SendEmail sends a notification; Message is its optional text.
	SendEmail bool
	Message   string
}

// InviteResult reports the per-user outcome.
type InviteResult struct {
	User    string `json:"User"`
	Status  bool   `json:"Status"`
	Message string `json:"Message"`
}

// Invite shares a resource with specific people.
func (s *Service) Invite(ctx context.Context, p InviteParams) ([]InviteResult, error) {
	if p.ResourceURL == "" || len(p.Recipients) == 0 {
		return nil, errors.New("sharing: resource URL and recipients are required")
	}
	var roles []map[string]any
	for _, r := range p.Recipients {
		role := r.Role
		if role == 0 {
			role = View
		}
		roles = append(roles, map[string]any{"UserId": r.Email, "Role": int(role)})
	}
	body := map[string]any{
		"resourceAddress":                     p.ResourceURL,
		"userRoleAssignments":                 roles,
		"validateExistingPermissions":         false,
		"additiveMode":                        true,
		"sendServerManagedNotification":       p.SendEmail,
		"customMessage":                       p.Message,
		"includeAnonymousLinksInNotification": false,
		"propagateAcl":                        false,
	}
	var out struct {
		Results []InviteResult `json:"value"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Post("SP.Sharing.DocumentSharingManager.UpdateDocumentSharingInfo", body), &out); err != nil {
		return nil, err
	}
	return out.Results, nil
}
