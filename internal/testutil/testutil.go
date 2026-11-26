// Package testutil provides a fake HTTP backend for resource package tests.
package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/14096/gopoint/httpx"
)

// Call is a recorded request.
type Call struct {
	Method string
	Path   string // decoded path
	Query  url.Values
	Header http.Header
	Body   string
}

// Server records requests and answers via the handler.
type Server struct {
	*httptest.Server
	Calls []Call
}

// Last returns the most recent call.
func (s *Server) Last() Call { return s.Calls[len(s.Calls)-1] }

// New starts a server; Client returns a generic client rooted at prefix.
func New(t *testing.T, h func(c Call, w http.ResponseWriter)) *Server {
	t.Helper()
	s := &Server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		c := Call{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Header: r.Header, Body: string(b)}
		s.Calls = append(s.Calls, c)
		h(c, w)
	}))
	t.Cleanup(s.Close)
	return s
}

// Client returns a client whose BaseURL is the server URL plus prefix.
func (s *Server) Client(prefix string) *httpx.Client {
	c, err := httpx.New(httpx.Config{
		BaseURL: s.URL + prefix, MaxAttempts: 1,
		Header: http.Header{"Accept": {"application/json;odata=nometadata"}},
	})
	if err != nil {
		panic(err) // test helper: the URL comes from httptest
	}
	return c
}
