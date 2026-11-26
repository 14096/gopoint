// Package users implements SharePoint site users and site groups.
package users

import (
	"context"
	"iter"
	"strconv"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes user and group operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// User is a site user.
type User struct {
	ID                int    `json:"Id"`
	Title             string `json:"Title"`
	LoginName         string `json:"LoginName"`
	Email             string `json:"Email"`
	IsSiteAdmin       bool   `json:"IsSiteAdmin"`
	PrincipalType     int    `json:"PrincipalType"`
	UserPrincipalName string `json:"UserPrincipalName"`
}

// Group is a SharePoint site group.
type Group struct {
	ID          int    `json:"Id"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	LoginName   string `json:"LoginName"`
	OwnerTitle  string `json:"OwnerTitle"`
}

// Current returns the user the client is authenticated as.
func (s *Service) Current(ctx context.Context) (*User, error) {
	return s.one(ctx, "web/currentuser")
}

// All lists the site users.
func (s *Service) All(ctx context.Context, q httpx.ODataQuery) ([]User, error) {
	return httpx.Collect[User](ctx, s.c, spx.Get("web/siteusers").WithQuery(q.Values()))
}

// Iter iterates over the site users.
func (s *Service) Iter(ctx context.Context, q httpx.ODataQuery) iter.Seq2[User, error] {
	return httpx.Paginate[User](ctx, s.c, spx.Get("web/siteusers").WithQuery(q.Values()))
}

// Get returns a site user by ID.
func (s *Service) Get(ctx context.Context, id int) (*User, error) {
	return s.one(ctx, "web/getuserbyid("+strconv.Itoa(id)+")")
}

// ByLoginName returns a site user by claims login name (e.g. i:0#.f|membership|a@b.c).
func (s *Service) ByLoginName(ctx context.Context, login string) (*User, error) {
	return s.one(ctx, "web/siteusers/getbyloginname("+spx.Quote(login)+")")
}

// ByEmail returns a site user by e-mail address.
func (s *Service) ByEmail(ctx context.Context, email string) (*User, error) {
	return s.one(ctx, "web/siteusers/getbyemail("+spx.Quote(email)+")")
}

// Ensure resolves a login name against the directory and adds it to the site's user list if needed.
func (s *Service) Ensure(ctx context.Context, login string) (*User, error) {
	var u User
	if err := httpx.JSON(ctx, s.c, spx.Post("web/ensureuser", map[string]string{"logonName": login}), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// Remove removes a user from the site's user list.
func (s *Service) Remove(ctx context.Context, id int) error {
	_, err := s.c.Do(ctx, spx.Post("web/siteusers/removebyid("+strconv.Itoa(id)+")", nil))
	return err
}

// Groups lists the groups a user belongs to.
func (s *Service) Groups(ctx context.Context, userID int) ([]Group, error) {
	return httpx.Collect[Group](ctx, s.c, spx.Get("web/getuserbyid("+strconv.Itoa(userID)+")/groups"))
}

// AllGroups lists the site groups.
func (s *Service) AllGroups(ctx context.Context) ([]Group, error) {
	return httpx.Collect[Group](ctx, s.c, spx.Get("web/sitegroups"))
}

// GetGroup returns a site group by ID.
func (s *Service) GetGroup(ctx context.Context, id int) (*Group, error) {
	var g Group
	if err := httpx.JSON(ctx, s.c, spx.Get("web/sitegroups("+strconv.Itoa(id)+")"), &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// GroupByName returns a site group by title.
func (s *Service) GroupByName(ctx context.Context, name string) (*Group, error) {
	var g Group
	if err := httpx.JSON(ctx, s.c, spx.Get("web/sitegroups/getbyname("+spx.Quote(name)+")"), &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// CreateGroup creates a site group.
func (s *Service) CreateGroup(ctx context.Context, title, description string) (*Group, error) {
	var g Group
	body := map[string]string{"Title": title, "Description": description}
	if err := httpx.JSON(ctx, s.c, spx.Post("web/sitegroups", body), &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// DeleteGroup deletes a site group.
func (s *Service) DeleteGroup(ctx context.Context, id int) error {
	_, err := s.c.Do(ctx, spx.Post("web/sitegroups/removebyid("+strconv.Itoa(id)+")", nil))
	return err
}

// GroupMembers lists the members of a group.
func (s *Service) GroupMembers(ctx context.Context, groupID int) ([]User, error) {
	return httpx.Collect[User](ctx, s.c, spx.Get("web/sitegroups("+strconv.Itoa(groupID)+")/users"))
}

// AddToGroup adds a user (by login name) to a group.
func (s *Service) AddToGroup(ctx context.Context, groupID int, login string) (*User, error) {
	var u User
	if err := httpx.JSON(ctx, s.c, spx.Post("web/sitegroups("+strconv.Itoa(groupID)+")/users", map[string]string{"LoginName": login}), &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// RemoveFromGroup removes a user from a group.
func (s *Service) RemoveFromGroup(ctx context.Context, groupID, userID int) error {
	_, err := s.c.Do(ctx, spx.Post("web/sitegroups("+strconv.Itoa(groupID)+")/users/removebyid("+strconv.Itoa(userID)+")", nil))
	return err
}

func (s *Service) one(ctx context.Context, path string) (*User, error) {
	var u User
	if err := httpx.JSON(ctx, s.c, spx.Get(path), &u); err != nil {
		return nil, err
	}
	return &u, nil
}
