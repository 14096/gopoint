// Package views implements SharePoint list views.
package views

import (
	"context"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes list view operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// View holds the commonly used view properties.
type View struct {
	ID                string `json:"Id"`
	Title             string `json:"Title"`
	ServerRelativeURL string `json:"ServerRelativeUrl"`
	ViewQuery         string `json:"ViewQuery"`
	RowLimit          int    `json:"RowLimit"`
	Paged             bool   `json:"Paged"`
	DefaultView       bool   `json:"DefaultView"`
	Hidden            bool   `json:"Hidden"`
	PersonalView      bool   `json:"PersonalView"`
	ViewType          string `json:"ViewType"`
	Scope             int    `json:"Scope"`
}

// CreateParams describes a new view.
type CreateParams struct {
	Title string
	// Fields are internal field names shown in the view.
	Fields []string
	// Query is the CAML <Where>/<OrderBy> fragment (without <Query>).
	Query    string
	RowLimit int
	Paged    bool
	// Personal creates a view visible only to the caller.
	Personal bool
	// DefaultView makes it the list's default view.
	DefaultView bool
}

// UpdateParams holds the changeable fields; nil values are left untouched.
type UpdateParams struct {
	Title       *string
	Query       *string
	RowLimit    *int
	Paged       *bool
	DefaultView *bool
	Hidden      *bool
}

func coll(l spx.ListRef) string { return l.Path() + "/views" }

func one(l spx.ListRef, id string) string { return coll(l) + "(guid" + spx.Quote(id) + ")" }

// All lists the views of a list.
func (s *Service) All(ctx context.Context, l spx.ListRef) ([]View, error) {
	return httpx.Collect[View](ctx, s.c, spx.Get(coll(l)))
}

// Get returns a view by ID.
func (s *Service) Get(ctx context.Context, l spx.ListRef, id string) (*View, error) {
	var v View
	if err := httpx.JSON(ctx, s.c, spx.Get(one(l, id)), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ByTitle returns a view by title.
func (s *Service) ByTitle(ctx context.Context, l spx.ListRef, title string) (*View, error) {
	var v View
	if err := httpx.JSON(ctx, s.c, spx.Get(coll(l)+"/getbytitle("+spx.Quote(title)+")"), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Create creates a view.
func (s *Service) Create(ctx context.Context, l spx.ListRef, p CreateParams) (*View, error) {
	body := map[string]any{"Title": p.Title, "Paged": p.Paged, "PersonalView": p.Personal, "SetAsDefaultView": p.DefaultView}
	if p.Query != "" {
		body["ViewQuery"] = p.Query
	}
	if p.RowLimit > 0 {
		body["RowLimit"] = p.RowLimit
	}
	if len(p.Fields) > 0 {
		body["ViewFields"] = p.Fields
	}
	var v View
	if err := httpx.JSON(ctx, s.c, spx.Post(coll(l), body), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Update changes a view.
func (s *Service) Update(ctx context.Context, l spx.ListRef, id string, p UpdateParams) error {
	body := map[string]any{}
	if p.Title != nil {
		body["Title"] = *p.Title
	}
	if p.Query != nil {
		body["ViewQuery"] = *p.Query
	}
	if p.RowLimit != nil {
		body["RowLimit"] = *p.RowLimit
	}
	if p.Paged != nil {
		body["Paged"] = *p.Paged
	}
	if p.DefaultView != nil {
		body["DefaultView"] = *p.DefaultView
	}
	if p.Hidden != nil {
		body["Hidden"] = *p.Hidden
	}
	_, err := s.c.Do(ctx, spx.Merge(one(l, id), body))
	return err
}

// Delete deletes a view.
func (s *Service) Delete(ctx context.Context, l spx.ListRef, id string) error {
	_, err := s.c.Do(ctx, spx.Remove(one(l, id)))
	return err
}

// Fields returns the internal names of the fields shown in a view, in order.
func (s *Service) Fields(ctx context.Context, l spx.ListRef, id string) ([]string, error) {
	var out struct {
		Items []string `json:"Items"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Get(one(l, id)+"/viewfields"), &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// AddField appends a field to a view.
func (s *Service) AddField(ctx context.Context, l spx.ListRef, id, field string) error {
	_, err := s.c.Do(ctx, spx.Post(one(l, id)+"/viewfields/addviewfield("+spx.Quote(field)+")", nil))
	return err
}

// RemoveField removes a field from a view.
func (s *Service) RemoveField(ctx context.Context, l spx.ListRef, id, field string) error {
	_, err := s.c.Do(ctx, spx.Post(one(l, id)+"/viewfields/removeviewfield("+spx.Quote(field)+")", nil))
	return err
}

// MoveField moves a field to a zero-based position.
func (s *Service) MoveField(ctx context.Context, l spx.ListRef, id, field string, index int) error {
	_, err := s.c.Do(ctx, spx.Post(one(l, id)+"/viewfields/moveviewfieldto", map[string]any{"field": field, "index": index}))
	return err
}

// RemoveAllFields empties a view's field list.
func (s *Service) RemoveAllFields(ctx context.Context, l spx.ListRef, id string) error {
	_, err := s.c.Do(ctx, spx.Post(one(l, id)+"/viewfields/removeallviewfields", nil))
	return err
}

// SetFields replaces the fields of a view with fields, in order.
func (s *Service) SetFields(ctx context.Context, l spx.ListRef, id string, fields []string) error {
	if err := s.RemoveAllFields(ctx, l, id); err != nil {
		return err
	}
	for _, f := range fields {
		if err := s.AddField(ctx, l, id, f); err != nil {
			return err
		}
	}
	return nil
}
