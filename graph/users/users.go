// Package users implements Graph directory users and groups.
package users

import (
	"context"
	"iter"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes user and group operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// User is a directory user.
type User struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle"`
}

// Group is a directory group (Microsoft 365 group, security group, ...).
type Group struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Mail        string `json:"mail"`
}

func get[T any](ctx context.Context, d httpx.Doer, path string) (*T, error) {
	var v T
	if err := httpx.JSON(ctx, d, httpx.NewRequest("GET", path), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Me returns the signed-in user (delegated credentials only).
func (s *Service) Me(ctx context.Context) (*User, error) { return get[User](ctx, s.c, "/me") }

// Get returns a user by ID or user principal name.
func (s *Service) Get(ctx context.Context, idOrUPN string) (*User, error) {
	return get[User](ctx, s.c, "/users/"+url.PathEscape(idOrUPN))
}

// All returns the users matching q.
func (s *Service) All(ctx context.Context, q httpx.ODataQuery) ([]User, error) {
	return httpx.Collect[User](ctx, s.c, httpx.NewRequest("GET", "/users").WithQuery(q.Values()))
}

// Iter iterates over users; use q.Filter, q.Select and q.Top to narrow the result.
func (s *Service) Iter(ctx context.Context, q httpx.ODataQuery) iter.Seq2[User, error] {
	return httpx.Paginate[User](ctx, s.c, httpx.NewRequest("GET", "/users").WithQuery(q.Values()))
}

// Groups iterates over groups.
func (s *Service) Groups(ctx context.Context, q httpx.ODataQuery) iter.Seq2[Group, error] {
	return httpx.Paginate[Group](ctx, s.c, httpx.NewRequest("GET", "/groups").WithQuery(q.Values()))
}

// GetGroup returns a group.
func (s *Service) GetGroup(ctx context.Context, id string) (*Group, error) {
	return get[Group](ctx, s.c, "/groups/"+url.PathEscape(id))
}

// MemberOf lists the groups a user belongs to.
func (s *Service) MemberOf(ctx context.Context, userID string) ([]Group, error) {
	return httpx.Collect[Group](ctx, s.c, httpx.NewRequest("GET", "/users/"+url.PathEscape(userID)+"/memberOf/microsoft.graph.group"))
}

// Members lists the user members of a group.
func (s *Service) Members(ctx context.Context, groupID string) ([]User, error) {
	return httpx.Collect[User](ctx, s.c, httpx.NewRequest("GET", "/groups/"+url.PathEscape(groupID)+"/members/microsoft.graph.user"))
}

// AddMember adds a user to a group.
func (s *Service) AddMember(ctx context.Context, groupID, userID string) error {
	ref := map[string]string{"@odata.id": "https://graph.microsoft.com/v1.0/directoryObjects/" + userID}
	_, err := s.c.Do(ctx, httpx.NewRequest("POST", "/groups/"+url.PathEscape(groupID)+"/members/$ref").WithJSON(ref))
	return err
}

// RemoveMember removes a user from a group.
func (s *Service) RemoveMember(ctx context.Context, groupID, userID string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", "/groups/"+url.PathEscape(groupID)+"/members/"+url.PathEscape(userID)+"/$ref"))
	return err
}
