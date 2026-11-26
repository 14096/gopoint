// Package spx holds helpers shared by the SharePoint resource packages: URL
// quoting, list/web addressing and the request shapes for POST/MERGE/DELETE.
package spx

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/14096/gopoint/httpx"
)

// JSONContentType is sent with request bodies.
const JSONContentType = "application/json;odata=nometadata"

// Quote renders s as an OData string literal, safe to embed in a URL path.
func Quote(s string) string {
	return "'" + url.PathEscape(strings.ReplaceAll(s, "'", "''")) + "'"
}

// QuotePath is like Quote but keeps '/' separators (for server-relative URLs).
func QuotePath(s string) string {
	var parts []string
	for p := range strings.SplitSeq(strings.ReplaceAll(s, "'", "''"), "/") {
		parts = append(parts, url.PathEscape(p))
	}
	return "'" + strings.Join(parts, "/") + "'"
}

// Scope is anything that owns collections such as fields or content types.
type Scope interface {
	// Path is the REST path of the object, relative to /_api.
	Path() string
}

type webScope struct{}

func (webScope) Path() string { return "web" }

// Web is the Scope of the current web.
var Web Scope = webScope{}

// ListRef identifies a list by title or ID (ID wins when both are set).
type ListRef struct {
	Title string
	ID    string
}

// ByTitle references a list by its title.
func ByTitle(title string) ListRef { return ListRef{Title: title} }

// ByID references a list by its GUID.
func ByID(id string) ListRef { return ListRef{ID: id} }

// Path implements Scope.
func (l ListRef) Path() string {
	if l.ID != "" {
		return "web/lists(guid" + Quote(l.ID) + ")"
	}
	return "web/lists/getbytitle(" + Quote(l.Title) + ")"
}

// Get builds a GET request.
func Get(path string) *httpx.Request { return httpx.NewRequest(http.MethodGet, path) }

// Post builds a POST request with an optional JSON body.
func Post(path string, body any) *httpx.Request {
	r := httpx.NewRequest(http.MethodPost, path)
	if body != nil {
		r.WithJSON(body).SetHeader("Content-Type", JSONContentType)
	}
	return r
}

// Merge builds a partial update (POST + X-HTTP-Method: MERGE).
func Merge(path string, body any) *httpx.Request {
	return Post(path, body).SetHeader("X-HTTP-Method", "MERGE").SetHeader("IF-MATCH", "*")
}

// Remove builds a delete (POST + X-HTTP-Method: DELETE).
func Remove(path string) *httpx.Request {
	return Post(path, nil).SetHeader("X-HTTP-Method", "DELETE").SetHeader("IF-MATCH", "*")
}

type pathScope string

func (p pathScope) Path() string { return string(p) }

// ItemRef is the Scope of a list item (by numeric ID); use it for item-level permissions and sharing.
func ItemRef(l ListRef, id int) Scope {
	return pathScope(l.Path() + "/items(" + strconv.Itoa(id) + ")")
}

// FileItem is the Scope of the list item behind a file (server-relative URL).
func FileItem(fileURL string) Scope {
	return pathScope("web/GetFileByServerRelativeUrl(" + QuotePath(fileURL) + ")/ListItemAllFields")
}

// FolderItem is the Scope of the list item behind a folder (server-relative URL).
func FolderItem(folderURL string) Scope {
	return pathScope("web/GetFolderByServerRelativeUrl(" + QuotePath(folderURL) + ")/ListItemAllFields")
}
