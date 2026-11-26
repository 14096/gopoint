// Package files implements the SharePoint file and folder resource, addressed by
// server-relative URL (e.g. /sites/team/Shared Documents/report.docx).
package files

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

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

// ChunkSize is the size of each piece sent by UploadLarge.
var ChunkSize int64 = 10 << 20

// SimpleUploadLimit is the largest size UploadLarge sends in one request.
var SimpleUploadLimit int64 = 4 << 20

// UploadLarge stores size bytes read from r as folderURL/name using
// SharePoint's chunked upload (StartUpload, ContinueUpload, FinishUpload),
// holding only one chunk in memory. Files up to SimpleUploadLimit use a single
// request. progress, when not nil, is called after each chunk with the bytes
// sent so far.
func (s *Service) UploadLarge(ctx context.Context, folderURL, name string, r io.ReaderAt, size int64, overwrite bool, progress func(sent, total int64)) (*File, error) {
	if size <= SimpleUploadLimit {
		buf, err := readChunk(r, 0, size)
		if err != nil {
			return nil, err
		}
		return s.Upload(ctx, folderURL, name, buf, overwrite)
	}
	// An empty file is created first; the chunks are then appended to it.
	f, err := s.Upload(ctx, folderURL, name, nil, overwrite)
	if err != nil {
		return nil, err
	}
	target := filePath(folderURL + "/" + name)
	if f.ServerRelativeURL != "" {
		target = filePath(f.ServerRelativeURL)
	}
	id := "guid" + spx.Quote(newGUID())
	for off := int64(0); off < size; {
		n := min(ChunkSize, size-off)
		buf, err := readChunk(r, off, n)
		if err != nil {
			return nil, err
		}
		var op string
		switch {
		case off == 0:
			op = "startupload(uploadId=" + id + ")"
		case off+n >= size:
			op = "finishupload(uploadId=" + id + ",fileOffset=" + strconv.FormatInt(off, 10) + ")"
		default:
			op = "continueupload(uploadId=" + id + ",fileOffset=" + strconv.FormatInt(off, 10) + ")"
		}
		req := httpx.NewRequest(http.MethodPost, target+"/"+op).WithBytes(buf, "application/octet-stream")
		resp, err := s.c.Do(ctx, req)
		if err != nil {
			s.cancelUpload(ctx, target, id)
			return nil, err
		}
		off += n
		if progress != nil {
			progress(off, size)
		}
		if off >= size {
			var out File
			if err := resp.JSON(&out); err != nil {
				return nil, err
			}
			return &out, nil
		}
	}
	return nil, errors.New("files: empty upload")
}

// cancelUpload is best effort: it frees the server-side upload session.
func (s *Service) cancelUpload(ctx context.Context, target, id string) {
	_, _ = s.c.Do(context.WithoutCancel(ctx), httpx.NewRequest(http.MethodPost, target+"/cancelupload(uploadId="+id+")"))
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

func newGUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
