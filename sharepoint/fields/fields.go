// Package fields implements the SharePoint "field" (column) resource for a web or a list.
package fields

import (
	"context"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes field operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// FieldTypeKind values for the most common column types.
const (
	Text     = 2
	Note     = 3
	DateTime = 4
	Choice   = 6
	Boolean  = 8
	Number   = 9
	Currency = 10
	URL      = 11
)

// Field holds the commonly used field properties.
type Field struct {
	ID            string `json:"Id"`
	Title         string `json:"Title"`
	InternalName  string `json:"InternalName"`
	StaticName    string `json:"StaticName"`
	TypeAsString  string `json:"TypeAsString"`
	FieldTypeKind int    `json:"FieldTypeKind"`
	Description   string `json:"Description"`
	Required      bool   `json:"Required"`
	Hidden        bool   `json:"Hidden"`
	ReadOnly      bool   `json:"ReadOnlyField"`
}

// List returns the fields of a web (spx.Web) or list (spx.ListRef).
func (s *Service) List(ctx context.Context, scope spx.Scope, q httpx.ODataQuery) ([]Field, error) {
	return httpx.Collect[Field](ctx, s.c, spx.Get(scope.Path()+"/fields").WithQuery(q.Values()))
}

// Get returns a field by internal name.
func (s *Service) Get(ctx context.Context, scope spx.Scope, internalName string) (*Field, error) {
	var f Field
	if err := httpx.JSON(ctx, s.c, spx.Get(scope.Path()+"/fields/getbyinternalnameortitle("+spx.Quote(internalName)+")"), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// CreateParams describes a new field. Extra holds additional properties
// (e.g. "Choices" for a choice column) merged into the request body.
type CreateParams struct {
	Title         string
	StaticName    string // becomes the internal name when set
	FieldTypeKind int
	Required      bool
	Description   string
	Extra         map[string]any
}

// Create adds a field.
func (s *Service) Create(ctx context.Context, scope spx.Scope, p CreateParams) (*Field, error) {
	body := map[string]any{}
	for k, v := range p.Extra {
		body[k] = v
	}
	body["Title"], body["FieldTypeKind"] = p.Title, p.FieldTypeKind
	body["Required"], body["Description"] = p.Required, p.Description
	if p.StaticName != "" {
		body["StaticName"] = p.StaticName
	}
	var f Field
	if err := httpx.JSON(ctx, s.c, spx.Post(scope.Path()+"/fields", body), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Update merges properties into the field.
func (s *Service) Update(ctx context.Context, scope spx.Scope, internalName string, props map[string]any) error {
	_, err := s.c.Do(ctx, spx.Merge(scope.Path()+"/fields/getbyinternalnameortitle("+spx.Quote(internalName)+")", props))
	return err
}

// Delete removes the field.
func (s *Service) Delete(ctx context.Context, scope spx.Scope, internalName string) error {
	_, err := s.c.Do(ctx, spx.Remove(scope.Path()+"/fields/getbyinternalnameortitle("+spx.Quote(internalName)+")"))
	return err
}
