// Package permissions implements SharePoint role assignments, role definitions
// and permission inheritance for webs, lists, items, files and folders.
package permissions

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes permission operations. Every method takes an spx.Scope:
// spx.Web, an spx.ListRef, spx.ItemRef, spx.FileItem or spx.FolderItem.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Principal is a user or group holding a role.
type Principal struct {
	ID            int    `json:"Id"`
	Title         string `json:"Title"`
	LoginName     string `json:"LoginName"`
	PrincipalType int    `json:"PrincipalType"`
}

// RoleDefinition is a permission level.
type RoleDefinition struct {
	ID           int    `json:"Id"`
	Name         string `json:"Name"`
	Description  string `json:"Description"`
	RoleTypeKind int    `json:"RoleTypeKind"`
}

// Assignment binds a principal to permission levels on an object.
type Assignment struct {
	PrincipalID int              `json:"PrincipalId"`
	Member      Principal        `json:"Member"`
	Roles       []RoleDefinition `json:"RoleDefinitionBindings"`
}

// BasePermissions is the High/Low bit mask returned by SharePoint.
type BasePermissions struct {
	High json.Number `json:"High"`
	Low  json.Number `json:"Low"`
}

// Assignments lists the role assignments of an object.
func (s *Service) Assignments(ctx context.Context, sc spx.Scope) ([]Assignment, error) {
	return httpx.Collect[Assignment](ctx, s.c,
		spx.Get(sc.Path()+"/roleassignments").SetQuery("$expand", "Member,RoleDefinitionBindings"))
}

// Grant gives a principal (user or group ID) a permission level (role definition ID).
func (s *Service) Grant(ctx context.Context, sc spx.Scope, principalID, roleDefID int) error {
	_, err := s.c.Do(ctx, spx.Post(sc.Path()+"/roleassignments/addroleassignment(principalid="+strconv.Itoa(principalID)+",roledefid="+strconv.Itoa(roleDefID)+")", nil))
	return err
}

// GrantByName is like Grant but resolves the permission level by name (e.g. "Edit", "Read").
func (s *Service) GrantByName(ctx context.Context, sc spx.Scope, principalID int, roleName string) error {
	r, err := s.RoleDefinitionByName(ctx, roleName)
	if err != nil {
		return err
	}
	return s.Grant(ctx, sc, principalID, r.ID)
}

// Revoke removes one permission level from a principal.
func (s *Service) Revoke(ctx context.Context, sc spx.Scope, principalID, roleDefID int) error {
	_, err := s.c.Do(ctx, spx.Post(sc.Path()+"/roleassignments/removeroleassignment(principalid="+strconv.Itoa(principalID)+",roledefid="+strconv.Itoa(roleDefID)+")", nil))
	return err
}

// RemovePrincipal removes all of a principal's assignments on the object.
func (s *Service) RemovePrincipal(ctx context.Context, sc spx.Scope, principalID int) error {
	_, err := s.c.Do(ctx, spx.Remove(sc.Path()+"/roleassignments/getbyprincipalid("+strconv.Itoa(principalID)+")"))
	return err
}

// HasUniquePermissions reports whether the object has its own permissions (inheritance broken).
func (s *Service) HasUniquePermissions(ctx context.Context, sc spx.Scope) (bool, error) {
	var out struct {
		Value bool `json:"value"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Get(sc.Path()+"/hasuniqueroleassignments"), &out); err != nil {
		return false, err
	}
	return out.Value, nil
}

// BreakInheritance stops inheriting permissions from the parent. With copy the
// current assignments are kept; with clearSubscopes child objects with unique
// permissions are reset too.
func (s *Service) BreakInheritance(ctx context.Context, sc spx.Scope, copy, clearSubscopes bool) error {
	p := "/breakroleinheritance(copyroleassignments=" + strconv.FormatBool(copy) + ",clearsubscopes=" + strconv.FormatBool(clearSubscopes) + ")"
	_, err := s.c.Do(ctx, spx.Post(sc.Path()+p, nil))
	return err
}

// ResetInheritance discards unique permissions and inherits from the parent again.
func (s *Service) ResetInheritance(ctx context.Context, sc spx.Scope) error {
	_, err := s.c.Do(ctx, spx.Post(sc.Path()+"/resetroleinheritance", nil))
	return err
}

// EffectivePermissions returns the permission mask a user (login name) has on the object.
func (s *Service) EffectivePermissions(ctx context.Context, sc spx.Scope, loginName string) (*BasePermissions, error) {
	var out BasePermissions
	req := spx.Get(sc.Path()+"/getusereffectivepermissions(@u)").SetQuery("@u", httpx.QuoteString(loginName))
	if err := httpx.JSON(ctx, s.c, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RoleDefinitions lists the permission levels defined on the web.
func (s *Service) RoleDefinitions(ctx context.Context) ([]RoleDefinition, error) {
	return httpx.Collect[RoleDefinition](ctx, s.c, spx.Get("web/roledefinitions"))
}

// RoleDefinitionByName returns a permission level by name (e.g. "Edit").
func (s *Service) RoleDefinitionByName(ctx context.Context, name string) (*RoleDefinition, error) {
	var r RoleDefinition
	if err := httpx.JSON(ctx, s.c, spx.Get("web/roledefinitions/getbyname("+spx.Quote(name)+")"), &r); err != nil {
		return nil, err
	}
	return &r, nil
}
