// Package folders implements Graph folder operations on drive items.
package folders

import (
	"context"
	"iter"

	"github.com/14096/gopoint/graph/drives"
	"github.com/14096/gopoint/graph/internal/gpath"
	"github.com/14096/gopoint/httpx"
)

// Service exposes folder operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

func (s *Service) do(ctx context.Context, method, path string, body any) (*drives.Item, error) {
	req := httpx.NewRequest(method, path)
	if body != nil {
		req.WithJSON(body)
	}
	var it drives.Item
	if err := httpx.JSON(ctx, s.c, req, &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// Get returns a folder; use "root" for the drive root.
func (s *Service) Get(ctx context.Context, drive, id string) (*drives.Item, error) {
	return s.do(ctx, "GET", gpath.Item(drive, id), nil)
}

// Iter iterates over the children of a folder.
func (s *Service) Iter(ctx context.Context, drive, id string) iter.Seq2[drives.Item, error] {
	return httpx.Paginate[drives.Item](ctx, s.c, httpx.NewRequest("GET", gpath.Item(drive, id)+"/children"))
}

// All returns the children of a folder.
func (s *Service) All(ctx context.Context, drive, id string) ([]drives.Item, error) {
	return httpx.Collect[drives.Item](ctx, s.c, httpx.NewRequest("GET", gpath.Item(drive, id)+"/children"))
}

// Create creates a folder named name under parentID. An existing name fails.
func (s *Service) Create(ctx context.Context, drive, parentID, name string) (*drives.Item, error) {
	body := map[string]any{"name": name, "folder": map[string]any{}, "@microsoft.graph.conflictBehavior": "fail"}
	return s.do(ctx, "POST", gpath.Item(drive, parentID)+"/children", body)
}

// Rename changes the name of a folder.
func (s *Service) Rename(ctx context.Context, drive, id, newName string) (*drives.Item, error) {
	return s.do(ctx, "PATCH", gpath.Item(drive, id), map[string]any{"name": newName})
}

// Move moves a folder under newParentID (optionally renaming it when newName is set).
func (s *Service) Move(ctx context.Context, drive, id, newParentID, newName string) (*drives.Item, error) {
	body := map[string]any{"parentReference": map[string]any{"id": newParentID}}
	if newName != "" {
		body["name"] = newName
	}
	return s.do(ctx, "PATCH", gpath.Item(drive, id), body)
}

// Copy starts an asynchronous copy and returns the monitor URL (poll it without authentication).
func (s *Service) Copy(ctx context.Context, drive, id, destDrive, destParentID, newName string) (string, error) {
	ref := map[string]any{"id": destParentID}
	if destDrive != "" {
		ref["driveId"] = destDrive
	}
	body := map[string]any{"parentReference": ref}
	if newName != "" {
		body["name"] = newName
	}
	resp, err := s.c.Do(ctx, httpx.NewRequest("POST", gpath.Item(drive, id)+"/copy").WithJSON(body))
	if err != nil {
		return "", err
	}
	return resp.Header.Get("Location"), nil
}

// Delete moves the folder to the recycle bin.
func (s *Service) Delete(ctx context.Context, drive, id string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", gpath.Item(drive, id)))
	return err
}

// DeletePermanently deletes the folder without using the recycle bin.
func (s *Service) DeletePermanently(ctx context.Context, drive, id string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("POST", gpath.Item(drive, id)+"/permanentDelete"))
	return err
}
