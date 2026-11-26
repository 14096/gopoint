package httpx

import (
	"context"
	"encoding/json"
	"iter"
)

// Paginate yields every item of a collection, following Graph
// (@odata.nextLink) and SharePoint (odata.nextLink, d.__next) next links. It
// understands the "value" (Graph, SharePoint nometadata) and "d.results"
// (SharePoint verbose) envelopes.
func Paginate[T any](ctx context.Context, d Doer, req *Request) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		cur := req.clone()
		for {
			if err := ctx.Err(); err != nil {
				yield(zero, err)
				return
			}
			resp, err := d.Do(ctx, cur)
			if err != nil {
				yield(zero, err)
				return
			}
			items, next, err := decodePage[T](resp.Body)
			if err != nil {
				yield(zero, err)
				return
			}
			for _, it := range items {
				if !yield(it, nil) {
					return
				}
			}
			if next == "" {
				return
			}
			n := cur.clone()
			n.Path, n.Query = next, nil
			cur = n
		}
	}
}

// Collect gathers all items of a paginated call.
func Collect[T any](ctx context.Context, d Doer, req *Request) ([]T, error) {
	var out []T
	for it, err := range Paginate[T](ctx, d, req) {
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func decodePage[T any](body []byte) ([]T, string, error) {
	var p struct {
		Value  []T    `json:"value"`
		Next   string `json:"@odata.nextLink"`
		NextSP string `json:"odata.nextLink"`
		D      *struct {
			Results []T    `json:"results"`
			Next    string `json:"__next"`
		} `json:"d"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, "", err
	}
	items, next := p.Value, p.Next
	if next == "" {
		next = p.NextSP
	}
	if p.D != nil {
		items = p.D.Results
		next = p.D.Next
	}
	return items, next, nil
}
