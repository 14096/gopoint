package httpx

import (
	"net/url"
	"strconv"
	"strings"
)

// ODataQuery holds the OData system query options shared by Graph and SharePoint.
type ODataQuery struct {
	Select  []string
	Expand  []string
	OrderBy []string
	Filter  string
	Top     int
}

// Values renders the query options.
func (q ODataQuery) Values() url.Values {
	v := url.Values{}
	if len(q.Select) > 0 {
		v.Set("$select", strings.Join(q.Select, ","))
	}
	if len(q.Expand) > 0 {
		v.Set("$expand", strings.Join(q.Expand, ","))
	}
	if len(q.OrderBy) > 0 {
		v.Set("$orderby", strings.Join(q.OrderBy, ","))
	}
	if q.Filter != "" {
		v.Set("$filter", q.Filter)
	}
	if q.Top > 0 {
		v.Set("$top", strconv.Itoa(q.Top))
	}
	return v
}

// QuoteString renders s as an OData string literal (” escapes a quote).
func QuoteString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
