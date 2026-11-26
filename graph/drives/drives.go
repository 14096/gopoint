// Package drives implements the Graph "drive" resource (document libraries and files).
package drives

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/url"
	"strings"

	"github.com/14096/gopoint/httpx"
)

// SimpleUploadLimit is the largest size sent in a single PUT; larger files use
// a resumable upload session. Graph's own limit for simple uploads is 4 MB.
var SimpleUploadLimit int64 = 4 << 20

// ChunkSize is the upload session chunk size; it must be a multiple of 320 KiB.
var ChunkSize int64 = 320 * 1024 * 12

// Service exposes drive operations.
type Service struct{ c httpx.Doer }

// New returns the service.
func New(c httpx.Doer) *Service { return &Service{c: c} }

// Drive is a document library.
type Drive struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DriveType string `json:"driveType"`
	WebURL    string `json:"webUrl"`
}

// Item is a file or folder.
type Item struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Size                 int64  `json:"size"`
	WebURL               string `json:"webUrl"`
	LastModifiedDateTime string `json:"lastModifiedDateTime"`
	Folder               *struct {
		ChildCount int `json:"childCount"`
	} `json:"folder"`
	File *struct {
		MimeType string `json:"mimeType"`
	} `json:"file"`
}

// IsFolder reports whether the item is a folder.
func (i Item) IsFolder() bool { return i.Folder != nil }

func driveBase(drive string) string { return "/drives/" + url.PathEscape(drive) }

// ForSite lists the drives (document libraries) of a site.
func (s *Service) ForSite(ctx context.Context, site string) ([]Drive, error) {
	return httpx.Collect[Drive](ctx, s.c, httpx.NewRequest("GET", "/sites/"+url.PathEscape(site)+"/drives"))
}

// Get returns a drive.
func (s *Service) Get(ctx context.Context, drive string) (*Drive, error) {
	var d Drive
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", driveBase(drive)), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Children iterates over the children of a folder; use "root" for the drive root.
func (s *Service) Children(ctx context.Context, drive, folderID string) iter.Seq2[Item, error] {
	return httpx.Paginate[Item](ctx, s.c, httpx.NewRequest("GET", driveBase(drive)+"/items/"+url.PathEscape(folderID)+"/children"))
}

// GetItem returns item metadata.
func (s *Service) GetItem(ctx context.Context, drive, itemID string) (*Item, error) {
	var it Item
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", driveBase(drive)+"/items/"+url.PathEscape(itemID)), &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// GetItemByPath returns item metadata by path relative to the drive root.
func (s *Service) GetItemByPath(ctx context.Context, drive, path string) (*Item, error) {
	var it Item
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("GET", driveBase(drive)+"/root:/"+escapePath(path)), &it); err != nil {
		return nil, err
	}
	return &it, nil
}

// Download returns the content of a file.
func (s *Service) Download(ctx context.Context, drive, itemID string) ([]byte, error) {
	r := httpx.NewRequest("GET", driveBase(drive)+"/items/"+url.PathEscape(itemID)+"/content").SetHeader("Accept", "*/*")
	resp, err := s.c.Do(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Delete deletes an item.
func (s *Service) Delete(ctx context.Context, drive, itemID string) error {
	_, err := s.c.Do(ctx, httpx.NewRequest("DELETE", driveBase(drive)+"/items/"+url.PathEscape(itemID)))
	return err
}

// Upload stores size bytes read from r as parentID/name (replacing an existing
// file). Files up to SimpleUploadLimit use one request; larger ones use a
// resumable upload session.
func (s *Service) Upload(ctx context.Context, drive, parentID, name string, r io.ReaderAt, size int64) (*Item, error) {
	target := driveBase(drive) + "/items/" + url.PathEscape(parentID) + ":/" + url.PathEscape(name) + ":"
	if size <= SimpleUploadLimit {
		buf, err := readChunk(r, 0, size)
		if err != nil {
			return nil, err
		}
		req := httpx.NewRequest("PUT", target+"/content").WithBytes(buf, "application/octet-stream")
		var it Item
		if err := httpx.JSON(ctx, s.c, req, &it); err != nil {
			return nil, err
		}
		return &it, nil
	}
	return s.uploadSession(ctx, target, r, size)
}

func (s *Service) uploadSession(ctx context.Context, target string, r io.ReaderAt, size int64) (*Item, error) {
	var sess struct {
		UploadURL string `json:"uploadUrl"`
	}
	body := map[string]any{"item": map[string]any{"@microsoft.graph.conflictBehavior": "replace"}}
	if err := httpx.JSON(ctx, s.c, httpx.NewRequest("POST", target+"/createUploadSession").WithJSON(body), &sess); err != nil {
		return nil, err
	}
	if sess.UploadURL == "" {
		return nil, errors.New("drives: upload session returned no uploadUrl")
	}
	for off := int64(0); off < size; off += ChunkSize {
		n := min(ChunkSize, size-off)
		buf, err := readChunk(r, off, n)
		if err != nil {
			return nil, err
		}
		req := httpx.NewRequest("PUT", sess.UploadURL).WithBytes(buf, "application/octet-stream").
			SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", off, off+n-1, size))
		req.NoAuth = true // the session URL is pre-authenticated
		resp, err := s.c.Do(ctx, req)
		if err != nil {
			return nil, err
		}
		if off+n >= size {
			var it Item
			if err := resp.JSON(&it); err != nil {
				return nil, err
			}
			return &it, nil
		}
	}
	return nil, errors.New("drives: empty upload")
}

func escapePath(p string) string {
	var segs []string
	for seg := range strings.SplitSeq(strings.Trim(p, "/"), "/") {
		segs = append(segs, url.PathEscape(seg))
	}
	return strings.Join(segs, "/")
}

func readChunk(r io.ReaderAt, off, n int64) ([]byte, error) {
	buf := make([]byte, n)
	got, err := r.ReadAt(buf, off)
	if int64(got) == n {
		return buf, nil
	}
	if err == nil || errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	return nil, err
}
