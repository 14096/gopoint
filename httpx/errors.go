package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Error is returned for non-2xx responses.
type Error struct {
	Status    int
	Code      string
	Message   string
	RequestID string
	Raw       []byte
}

func (e *Error) Error() string {
	s := fmt.Sprintf("http %d", e.Status)
	if e.Code != "" {
		s += " " + e.Code
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	if e.RequestID != "" {
		s += " (request-id " + e.RequestID + ")"
	}
	return s
}

// IsNotFound reports whether err is an *Error with status 404.
func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

// NewError builds an *Error from a non-2xx response, for callers that receive
// statuses outside Client.Do (for example $batch sub-responses).
func NewError(status int, h http.Header, body []byte) *Error { return parseError(status, h, body) }

// parseError understands Graph ({"error":{"code","message":"..."}}), SharePoint
// verbose ({"error":{"code","message":{"value"}}}) and SharePoint nometadata
// ({"odata.error":{...}}) bodies.
func parseError(status int, h http.Header, body []byte) *Error {
	e := &Error{Status: status, Raw: body}
	for _, k := range []string{"request-id", "SPRequestGuid", "client-request-id"} {
		if v := h.Get(k); v != "" {
			e.RequestID = v
			break
		}
	}
	var env map[string]json.RawMessage
	if json.Unmarshal(body, &env) != nil {
		e.Message = strings.TrimSpace(string(body))
		return e
	}
	raw, ok := env["error"]
	if !ok {
		raw, ok = env["odata.error"]
	}
	if !ok {
		return e
	}
	var inner struct {
		Code    string          `json:"code"`
		Message json.RawMessage `json:"message"`
	}
	if json.Unmarshal(raw, &inner) != nil {
		return e
	}
	e.Code = inner.Code
	var s string
	var obj struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(inner.Message, &s) == nil {
		e.Message = s
	} else if json.Unmarshal(inner.Message, &obj) == nil {
		e.Message = obj.Value
	}
	return e
}
