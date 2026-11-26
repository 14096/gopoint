package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
)

// Request describes a call. Path is relative to the client's BaseURL, or an
// absolute URL (which must be on the BaseURL host unless NoAuth is set).
type Request struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	// Body is fully buffered so it can be replayed on retry.
	Body []byte
	// NoAuth skips the Authorization header and the host check; use it for
	// pre-authenticated URLs such as upload sessions.
	NoAuth bool
	// NoPrepare skips the client's Prepare hook.
	NoPrepare bool

	err error // deferred construction error, reported by Client.Do
}

// NewRequest returns a request for the given method and path.
func NewRequest(method, path string) *Request {
	return &Request{Method: method, Path: path}
}

// SetQuery sets a query parameter and returns r.
func (r *Request) SetQuery(k, v string) *Request {
	if r.Query == nil {
		r.Query = url.Values{}
	}
	r.Query.Set(k, v)
	return r
}

// SetHeader sets a header and returns r.
func (r *Request) SetHeader(k, v string) *Request {
	if r.Header == nil {
		r.Header = http.Header{}
	}
	r.Header.Set(k, v)
	return r
}

// WithQuery merges values into the query and returns r.
func (r *Request) WithQuery(q url.Values) *Request {
	if r.Query == nil {
		r.Query = url.Values{}
	}
	for k, vs := range q {
		r.Query[k] = slices.Clone(vs)
	}
	return r
}

// WithBytes sets a raw body with the given content type.
func (r *Request) WithBytes(b []byte, contentType string) *Request {
	r.Body = b
	if contentType != "" {
		r.SetHeader("Content-Type", contentType)
	}
	return r
}

// WithJSON marshals v as the body. Errors surface when the request is sent.
func (r *Request) WithJSON(v any) *Request {
	b, err := json.Marshal(v)
	if err != nil {
		r.Body = nil
		r.err = fmt.Errorf("httpx: marshal body: %w", err)
		return r
	}
	return r.WithBytes(b, "application/json")
}

// Apply runs the options against r and returns r.
func (r *Request) Apply(opts ...Option) *Request {
	for _, o := range opts {
		if o != nil {
			o(r)
		}
	}
	return r
}

// NewRequestFromHTTP converts a standard library request. The body is read
// and buffered; the original request is not otherwise modified.
func NewRequestFromHTTP(hr *http.Request) (*Request, error) {
	r := &Request{Method: hr.Method, Header: hr.Header.Clone()}
	u := *hr.URL
	if u.Host == "" {
		r.Path = u.Path
		r.Query = u.Query()
	} else {
		r.Query = u.Query()
		u.RawQuery = ""
		r.Path = u.String()
	}
	if hr.Body != nil {
		defer hr.Body.Close()
		b, err := io.ReadAll(hr.Body)
		if err != nil {
			return nil, fmt.Errorf("httpx: read body: %w", err)
		}
		r.Body = b
	}
	return r, nil
}

func (r *Request) clone() *Request {
	c := *r
	c.Header = r.Header.Clone()
	return &c
}

// Response is a fully read HTTP response.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// ETag returns the ETag response header, or "".
func (r *Response) ETag() string { return r.Header.Get("ETag") }

// JSON decodes the body into out. Empty bodies are ignored.
func (r *Response) JSON(out any) error {
	if out == nil || len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, out)
}
