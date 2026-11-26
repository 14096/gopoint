// Package search implements the SharePoint Search REST API.
package search

import (
	"context"
	"iter"
	"strconv"
	"strings"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes search operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Query describes a search request.
type Query struct {
	Text              string   // KQL query text
	SelectProperties  []string // managed properties to return
	RefinementFilters []string
	SortList          []Sort
	RowLimit          int // page size (default 50, SharePoint max 500)
	StartRow          int
	TrimDuplicates    bool
	SourceID          string
	Culture           int
}

// Sort orders results by a managed property.
type Sort struct {
	Property   string
	Descending bool
}

// Row is one result: managed property name to value.
type Row map[string]string

// Result is one page of results.
type Result struct {
	Rows      []Row
	RowCount  int
	TotalRows int
	// TotalRowsIncludingDuplicates is the count before duplicate trimming.
	TotalRowsIncludingDuplicates int
}

func (q Query) values() map[string]string {
	v := map[string]string{"querytext": httpx.QuoteString(q.Text)}
	if len(q.SelectProperties) > 0 {
		v["selectproperties"] = httpx.QuoteString(strings.Join(q.SelectProperties, ","))
	}
	if len(q.RefinementFilters) > 0 {
		v["refinementfilters"] = httpx.QuoteString(strings.Join(q.RefinementFilters, ","))
	}
	if len(q.SortList) > 0 {
		var parts []string
		for _, s := range q.SortList {
			dir := "0"
			if s.Descending {
				dir = "1"
			}
			parts = append(parts, s.Property+":"+dir)
		}
		v["sortlist"] = httpx.QuoteString(strings.Join(parts, ","))
	}
	if q.RowLimit > 0 {
		v["rowlimit"] = strconv.Itoa(q.RowLimit)
	}
	if q.StartRow > 0 {
		v["startrow"] = strconv.Itoa(q.StartRow)
	}
	if q.TrimDuplicates {
		v["trimduplicates"] = "true"
	} else {
		v["trimduplicates"] = "false"
	}
	if q.SourceID != "" {
		v["sourceid"] = httpx.QuoteString(q.SourceID)
	}
	if q.Culture > 0 {
		v["culture"] = strconv.Itoa(q.Culture)
	}
	return v
}

type response struct {
	PrimaryQueryResult struct {
		RelevantResults struct {
			RowCount                     int `json:"RowCount"`
			TotalRows                    int `json:"TotalRows"`
			TotalRowsIncludingDuplicates int `json:"TotalRowsIncludingDuplicates"`
			Table                        struct {
				Rows []struct {
					Cells []struct {
						Key   string `json:"Key"`
						Value string `json:"Value"`
					} `json:"Cells"`
				} `json:"Rows"`
			} `json:"Table"`
		} `json:"RelevantResults"`
	} `json:"PrimaryQueryResult"`
}

// Do runs one query page.
func (s *Service) Do(ctx context.Context, q Query) (*Result, error) {
	req := spx.Get("search/query")
	for k, v := range q.values() {
		req.SetQuery(k, v)
	}
	var out response
	if err := httpx.JSON(ctx, s.c, req, &out); err != nil {
		return nil, err
	}
	rr := out.PrimaryQueryResult.RelevantResults
	res := &Result{RowCount: rr.RowCount, TotalRows: rr.TotalRows, TotalRowsIncludingDuplicates: rr.TotalRowsIncludingDuplicates}
	for _, r := range rr.Table.Rows {
		row := make(Row, len(r.Cells))
		for _, c := range r.Cells {
			row[c.Key] = c.Value
		}
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

// Iter iterates over all results, fetching pages as needed.
func (s *Service) Iter(ctx context.Context, q Query) iter.Seq2[Row, error] {
	return func(yield func(Row, error) bool) {
		if q.RowLimit <= 0 {
			q.RowLimit = 50
		}
		for {
			res, err := s.Do(ctx, q)
			if err != nil {
				yield(nil, err)
				return
			}
			for _, r := range res.Rows {
				if !yield(r, nil) {
					return
				}
			}
			q.StartRow += len(res.Rows)
			if len(res.Rows) == 0 || q.StartRow >= res.TotalRows {
				return
			}
		}
	}
}

// Suggest returns query suggestions for a prefix.
func (s *Service) Suggest(ctx context.Context, text string) ([]string, error) {
	req := spx.Get("search/suggest").SetQuery("querytext", httpx.QuoteString(text))
	var out struct {
		Queries []struct {
			Query string `json:"Query"`
		} `json:"Queries"`
	}
	if err := httpx.JSON(ctx, s.c, req, &out); err != nil {
		return nil, err
	}
	var res []string
	for _, q := range out.Queries {
		res = append(res, q.Query)
	}
	return res, nil
}
