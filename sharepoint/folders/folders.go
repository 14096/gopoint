// Package folders implements SharePoint folder operations, addressed by
// server-relative URL (e.g. /sites/team/Shared Documents/Reports).
package folders

import (
	"context"
	"iter"
	"time"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes folder operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Folder holds the commonly used folder properties.
type Folder struct {
	Name              string    `json:"Name"`
	ServerRelativeURL string    `json:"ServerRelativeUrl"`
	ItemCount         int       `json:"ItemCount"`
	UniqueID          string    `json:"UniqueId"`
	TimeCreated       time.Time `json:"TimeCreated"`
	TimeLastModified  time.Time `json:"TimeLastModified"`
}

func path(folderURL string) string {
	return "web/GetFolderByServerRelativeUrl(" + spx.QuotePath(folderURL) + ")"
}

// Get returns a folder.
func (s *Service) Get(ctx context.Context, folderURL string) (*Folder, error) {
	var f Folder
	if err := httpx.JSON(ctx, s.c, spx.Get(path(folderURL)), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Exists reports whether the folder exists.
func (s *Service) Exists(ctx context.Context, folderURL string) (bool, error) {
	var out struct {
		Value bool `json:"value"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Get(path(folderURL)+"/Exists"), &out); err != nil {
		return false, err
	}
	return out.Value, nil
}

// Iter iterates over the subfolders of a folder.
func (s *Service) Iter(ctx context.Context, folderURL string) iter.Seq2[Folder, error] {
	return httpx.Paginate[Folder](ctx, s.c, spx.Get(path(folderURL)+"/Folders"))
}

// All returns the subfolders of a folder.
func (s *Service) All(ctx context.Context, folderURL string) ([]Folder, error) {
	return httpx.Collect[Folder](ctx, s.c, spx.Get(path(folderURL)+"/Folders"))
}

// Create creates a folder (parents must exist).
func (s *Service) Create(ctx context.Context, folderURL string) (*Folder, error) {
	var f Folder
	if err := httpx.JSON(ctx, s.c, spx.Post("web/folders/add("+spx.QuotePath(folderURL)+")", nil), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Rename changes the folder name (not its location).
func (s *Service) Rename(ctx context.Context, folderURL, newName string) error {
	body := map[string]any{"FileLeafRef": newName, "Title": newName}
	_, err := s.c.Do(ctx, spx.Merge(path(folderURL)+"/ListItemAllFields", body))
	return err
}

// Move moves the folder to newURL.
func (s *Service) Move(ctx context.Context, folderURL, newURL string) error {
	_, err := s.c.Do(ctx, spx.Post(path(folderURL)+"/moveto(newurl="+spx.QuotePath(newURL)+",flags=1)", nil))
	return err
}

// Delete permanently deletes the folder.
func (s *Service) Delete(ctx context.Context, folderURL string) error {
	_, err := s.c.Do(ctx, spx.Remove(path(folderURL)))
	return err
}

// Recycle moves the folder to the recycle bin and returns the recycle bin item ID.
func (s *Service) Recycle(ctx context.Context, folderURL string) (string, error) {
	var out struct {
		Value string `json:"value"`
	}
	if err := httpx.JSON(ctx, s.c, spx.Post(path(folderURL)+"/recycle", nil), &out); err != nil {
		return "", err
	}
	return out.Value, nil
}
