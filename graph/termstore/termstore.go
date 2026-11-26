// Package termstore implements the Graph term store (managed metadata) of a site.
package termstore

import (
	"context"
	"iter"
	"net/url"

	"github.com/14096/gopoint/httpx"
)

// Service exposes term store operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Group is a term group.
type Group struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
}

// Set is a term set.
type Set struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	CreatedDateTime string `json:"createdDateTime"`
	Localized       []struct {
		Name        string `json:"name"`
		LanguageTag string `json:"languageTag"`
		IsDefault   bool   `json:"isDefault"`
	} `json:"localizedNames"`
}

// Label is a localized term name.
type Label struct {
	Name        string `json:"name"`
	LanguageTag string `json:"languageTag"`
	IsDefault   bool   `json:"isDefault"`
}

// Term is a term in a set.
type Term struct {
	ID              string  `json:"id"`
	Labels          []Label `json:"labels"`
	CreatedDateTime string  `json:"createdDateTime"`
	Descriptions    []struct {
		Description string `json:"description"`
		LanguageTag string `json:"languageTag"`
	} `json:"descriptions"`
}

func base(site string) string { return "/sites/" + url.PathEscape(site) + "/termStore" }

func setPath(site, set string) string { return base(site) + "/sets/" + url.PathEscape(set) }

// Groups iterates over the term groups.
func (s *Service) Groups(ctx context.Context, site string) iter.Seq2[Group, error] {
	return httpx.Paginate[Group](ctx, s.c, httpx.NewRequest("GET", base(site)+"/groups"))
}

// Sets iterates over the term sets of a group.
func (s *Service) Sets(ctx context.Context, site, group string) iter.Seq2[Set, error] {
	return httpx.Paginate[Set](ctx, s.c, httpx.NewRequest("GET", base(site)+"/groups/"+url.PathEscape(group)+"/sets"))
}

// GetSet returns a term set.
func (s *Service) GetSet(ctx context.Context, site, set string) (*Set, error) {
	var out Set
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", setPath(site, set)), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Terms iterates over the top-level terms of a set.
func (s *Service) Terms(ctx context.Context, site, set string) iter.Seq2[Term, error] {
	return httpx.Paginate[Term](ctx, s.c, httpx.NewRequest("GET", setPath(site, set)+"/children"))
}

// Children iterates over the child terms of a term.
func (s *Service) Children(ctx context.Context, site, set, term string) iter.Seq2[Term, error] {
	return httpx.Paginate[Term](ctx, s.c, httpx.NewRequest("GET", setPath(site, set)+"/terms/"+url.PathEscape(term)+"/children"))
}

// GetTerm returns a term.
func (s *Service) GetTerm(ctx context.Context, site, set, term string) (*Term, error) {
	var out Term
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", setPath(site, set)+"/terms/"+url.PathEscape(term)), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateTerm adds a top-level term, or a child when parent is not empty.
func (s *Service) CreateTerm(ctx context.Context, site, set, parent string, labels []Label) (*Term, error) {
	path := setPath(site, set) + "/children"
	if parent != "" {
		path = setPath(site, set) + "/terms/" + url.PathEscape(parent) + "/children"
	}
	var out Term
	r := httpx.NewRequest("POST", path).WithJSON(map[string]any{"labels": labels})
	if err := httpx.JSON(ctx, s.c, r, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteTerm deletes a term.
func (s *Service) DeleteTerm(ctx context.Context, site, set, term string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", setPath(site, set)+"/terms/"+url.PathEscape(term)))
	return err
}
