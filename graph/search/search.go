// Package search implements the Microsoft Search API (POST /search/query).
package search

import (
	"context"
	"encoding/json"
	"errors"
	"iter"

	"github.com/14096/gopoint/httpx"
)

// Service exposes search operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Query describes a search request.
type Query struct {
	// EntityTypes are e.g. "driveItem", "listItem", "site", "list", "drive", "message".
	EntityTypes []string
	Text        string // KQL query string
	From        int
	Size        int // page size (max 500)
	Fields      []string
	// Region is required for application permissions (e.g. "EUR").
	Region string
}

// Hit is one search result. Resource is the raw entity (driveItem, listItem, ...).
type Hit struct {
	ID       string          `json:"hitId"`
	Rank     int             `json:"rank"`
	Summary  string          `json:"summary"`
	Resource json.RawMessage `json:"resource"`
}

// Decode unmarshals the hit's resource into v (e.g. a *drives.Item).
func (h Hit) Decode(v any) error { return json.Unmarshal(h.Resource, v) }

// Result is one page of hits.
type Result struct {
	Hits          []Hit
	Total         int
	MoreAvailable bool
}

// Do runs one query page.
func (s *Service) Do(ctx context.Context, q Query) (*Result, error) {
	if len(q.EntityTypes) == 0 || q.Text == "" {
		return nil, errors.New("search: entity types and query text are required")
	}
	r := map[string]any{"entityTypes": q.EntityTypes, "query": map[string]any{"queryString": q.Text}, "from": q.From}
	if q.Size > 0 {
		r["size"] = q.Size
	}
	if len(q.Fields) > 0 {
		r["fields"] = q.Fields
	}
	if q.Region != "" {
		r["region"] = q.Region
	}
	var out struct {
		Value []struct {
			HitsContainers []struct {
				Hits                 []Hit `json:"hits"`
				Total                int   `json:"total"`
				MoreResultsAvailable bool  `json:"moreResultsAvailable"`
			} `json:"hitsContainers"`
		} `json:"value"`
	}
	req := httpx.NewRequest("POST", "/search/query").WithJSON(map[string]any{"requests": []any{r}})
	if err := httpx.JSON(ctx, s.c, req, &out); err != nil {
		return nil, err
	}
	res := &Result{}
	for _, v := range out.Value {
		for _, hc := range v.HitsContainers {
			res.Hits = append(res.Hits, hc.Hits...)
			res.Total += hc.Total
			res.MoreAvailable = res.MoreAvailable || hc.MoreResultsAvailable
		}
	}
	return res, nil
}

// Iter iterates over all hits, fetching pages as needed.
func (s *Service) Iter(ctx context.Context, q Query) iter.Seq2[Hit, error] {
	return func(yield func(Hit, error) bool) {
		if q.Size <= 0 {
			q.Size = 25
		}
		for {
			res, err := s.Do(ctx, q)
			if err != nil {
				yield(Hit{}, err)
				return
			}
			for _, h := range res.Hits {
				if !yield(h, nil) {
					return
				}
			}
			if !res.MoreAvailable || len(res.Hits) == 0 {
				return
			}
			q.From += len(res.Hits)
		}
	}
}
