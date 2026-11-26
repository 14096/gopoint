package httpx

import (
	"context"
	"encoding/json"
)

// Delta follows a Graph delta query to the end and returns every item plus the
// final @odata.deltaLink. Store the link and pass it as req.Path (an absolute
// URL on the client's host) next time to receive only changes.
func Delta[T any](ctx context.Context, d Doer, req *Request) ([]T, string, error) {
	var out []T
	cur := req.clone()
	for {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		resp, err := d.Do(ctx, cur)
		if err != nil {
			return nil, "", err
		}
		var p struct {
			Value []T    `json:"value"`
			Next  string `json:"@odata.nextLink"`
			Delta string `json:"@odata.deltaLink"`
		}
		if err := json.Unmarshal(resp.Body, &p); err != nil {
			return nil, "", err
		}
		out = append(out, p.Value...)
		if p.Next == "" {
			return out, p.Delta, nil
		}
		n := cur.clone()
		n.Path, n.Query = p.Next, nil
		cur = n
	}
}
