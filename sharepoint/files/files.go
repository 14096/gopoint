// Package files implements the SharePoint file and folder resource, addressed by
// server-relative URL (e.g. /sites/team/Shared Documents/report.docx).
package files

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// Service exposes file and folder operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// File holds the commonly used file properties.
type File struct {
	Name              string      `json:"Name"`
	ServerRelativeURL string      `json:"ServerRelativeUrl"`
	Length            json.Number `json:"Length"`
	TimeCreated       string      `json:"TimeCreated"`
	TimeLastModified  string      `json:"TimeLastModified"`
}

// Folder holds the commonly used folder properties.
type Folder struct {
	Name              string `json:"Name"`
	ServerRelativeURL string `json:"ServerRelativeUrl"`
	ItemCount         int    `json:"ItemCount"`
}

func folderPath(url string) string {
	return "web/GetFolderByServerRelativeUrl(" + spx.QuotePath(url) + ")"
}
func filePath(url string) string { return "web/GetFileByServerRelativeUrl(" + spx.QuotePath(url) + ")" }

// ListFiles returns the files directly inside a folder.
func (s *Service) ListFiles(ctx context.Context, folderURL string) ([]File, error) {
	return httpx.Collect[File](ctx, s.c, spx.Get(folderPath(folderURL)+"/Files"))
}

// ListFolders returns the subfolders directly inside a folder.
func (s *Service) ListFolders(ctx context.Context, folderURL string) ([]Folder, error) {
	return httpx.Collect[Folder](ctx, s.c, spx.Get(folderPath(folderURL)+"/Folders"))
}

// CreateFolder creates a folder.
func (s *Service) CreateFolder(ctx context.Context, folderURL string) (*Folder, error) {
	var f Folder
	if err := httpx.JSON(ctx, s.c, spx.Post("web/folders/add("+spx.QuotePath(folderURL)+")", nil), &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// DeleteFolder deletes a folder.
func (s *Service) DeleteFolder(ctx context.Context, folderURL string) error {
	_, err := s.c.Do(ctx, spx.Remove(folderPath(folderURL)))
	return err
}

// Upload writes data to folderURL/name. It uses the single-request upload,
// which SharePoint limits to 250 MB; the whole content is held in memory.
func (s *Service) Upload(ctx context.Context, folderURL, name string, data []byte, overwrite bool) (*File, error) {
	ow := "false"
	if overwrite {
		ow = "true"
	}
	req := httpx.NewRequest(http.MethodPost, folderPath(folderURL)+"/Files/add(url="+spx.Quote(name)+",overwrite="+ow+")")
	req.WithBytes(data, "application/octet-stream")
	var f File
	if err := httpx.JSON(ctx, s.c, req, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Download returns the content of a file.
func (s *Service) Download(ctx context.Context, fileURL string) ([]byte, error) {
	req := spx.Get(filePath(fileURL)+"/$value").SetHeader("Accept", "*/*")
	resp, err := s.c.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// DeleteFile deletes a file.
func (s *Service) DeleteFile(ctx context.Context, fileURL string) error {
	_, err := s.c.Do(ctx, spx.Remove(filePath(fileURL)))
	return err
}
