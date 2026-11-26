package httpx

import "net/url"

// Option adjusts a single request; pass them to the Client helpers or to
// Request.Apply.
type Option func(*Request)

// WithHeader sets a header for this call only.
func WithHeader(k, v string) Option {
	return func(r *Request) { r.SetHeader(k, v) }
}

// WithQuery merges query parameters for this call only.
func WithQuery(q url.Values) Option {
	return func(r *Request) { r.WithQuery(q) }
}

// WithOData merges OData system query options for this call only.
func WithOData(q ODataQuery) Option {
	return func(r *Request) { r.WithQuery(q.Values()) }
}

// WithIfMatch sets If-Match for optimistic concurrency. Use "*" to match any
// version.
func WithIfMatch(etag string) Option {
	return WithHeader("If-Match", etag)
}

// WithIfNoneMatch sets If-None-Match, e.g. for conditional GETs.
func WithIfNoneMatch(etag string) Option {
	return WithHeader("If-None-Match", etag)
}

// WithNoAuth skips the Authorization header and host check, for
// pre-authenticated URLs.
func WithNoAuth() Option {
	return func(r *Request) { r.NoAuth = true }
}
