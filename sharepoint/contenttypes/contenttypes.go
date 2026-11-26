// Package contenttypes implements the SharePoint "content type" resource for a web or a list.
package contenttypes

import (
	"context"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes content type operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// ContentType holds the commonly used properties.
type ContentType struct {
	ID          string `json:"StringId"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Group       string `json:"Group"`
	Hidden      bool   `json:"Hidden"`
}

// List returns the content types of a web (spx.Web) or list (spx.ListRef).
func (s *Service) List(ctx context.Context, scope spx.Scope) ([]ContentType, error) {
	return httpx.Collect[ContentType](ctx, s.c, spx.Get(scope.Path()+"/contenttypes"))
}

// Get returns a content type by ID (e.g. "0x0101").
func (s *Service) Get(ctx context.Context, scope spx.Scope, id string) (*ContentType, error) {
	var ct ContentType
	if err := httpx.JSON(ctx, s.c, spx.Get(scope.Path()+"/contenttypes("+spx.Quote(id)+")"), &ct); err != nil {
		return nil, err
	}
	return &ct, nil
}

// CreateParams describes a new content type.
type CreateParams struct {
	// ID is the full content type ID, which must start with the parent's ID.
	ID          string
	Name        string
	Description string
	Group       string
}

// Create adds a content type.
func (s *Service) Create(ctx context.Context, scope spx.Scope, p CreateParams) (*ContentType, error) {
	body := map[string]any{
		"Id": map[string]string{"StringValue": p.ID}, "Name": p.Name,
		"Description": p.Description, "Group": p.Group,
	}
	var ct ContentType
	if err := httpx.JSON(ctx, s.c, spx.Post(scope.Path()+"/contenttypes", body), &ct); err != nil {
		return nil, err
	}
	return &ct, nil
}

// Delete removes a content type.
func (s *Service) Delete(ctx context.Context, scope spx.Scope, id string) error {
	_, err := s.c.Do(ctx, spx.Remove(scope.Path()+"/contenttypes("+spx.Quote(id)+")"))
	return err
}
