package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/14096/gopoint/httpx"
)

// MaxBatchSize is the number of requests Graph accepts in one $batch call.
const MaxBatchSize = 20

// BatchRequest is one call inside a batch.
type BatchRequest struct {
	// ID is optional; positions in the input slice are used when empty.
	ID     string
	Method string
	// URL is relative to the Graph base URL, e.g. "/sites/root".
	URL     string
	Headers map[string]string
	// Body is JSON encoded when not nil.
	Body any
}

// BatchResponse is the result of one call inside a batch.
type BatchResponse struct {
	ID      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

// Err returns an *httpx.Error when the sub-request did not succeed.
func (r BatchResponse) Err() error {
	if r.Status >= 200 && r.Status < 300 {
		return nil
	}
	return httpx.NewError(r.Status, http.Header{}, r.Body)
}

// JSON decodes the sub-response body into out; empty bodies are ignored.
func (r BatchResponse) JSON(out any) error {
	if out == nil || len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, out)
}

type batchItem struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

// Batch sends the requests through Graph's $batch endpoint, splitting them into
// groups of MaxBatchSize. Results are returned in input order. A failing
// sub-request does not fail the call: check BatchResponse.Err for each.
func (c *Client) Batch(ctx context.Context, reqs []BatchRequest) ([]BatchResponse, error) {
	out := make([]BatchResponse, len(reqs))
	for start := 0; start < len(reqs); start += MaxBatchSize {
		end := min(start+MaxBatchSize, len(reqs))
		items := make([]batchItem, 0, end-start)
		for i := start; i < end; i++ {
			r := reqs[i]
			it := batchItem{ID: strconv.Itoa(i), Method: r.Method, URL: r.URL, Headers: r.Headers, Body: r.Body}
			if it.Method == "" {
				it.Method = http.MethodGet
			}
			if r.Body != nil && it.Headers["Content-Type"] == "" {
				h := make(map[string]string, len(r.Headers)+1)
				for k, v := range r.Headers {
					h[k] = v
				}
				h["Content-Type"] = "application/json"
				it.Headers = h
			}
			items = append(items, it)
		}
		var res struct {
			Responses []BatchResponse `json:"responses"`
		}
		if err := c.raw.Post(ctx, "/$batch", map[string]any{"requests": items}, &res); err != nil {
			return nil, err
		}
		for _, br := range res.Responses {
			i, err := strconv.Atoi(br.ID)
			if err != nil || i < start || i >= end {
				return nil, fmt.Errorf("graph: batch response with unexpected id %q", br.ID)
			}
			if reqs[i].ID != "" {
				br.ID = reqs[i].ID
			}
			out[i] = br
		}
	}
	return out, nil
}
