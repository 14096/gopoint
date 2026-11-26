package sharepoint

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/14096/gopoint/httpx"
	"github.com/14096/gopoint/sharepoint/spx"
)

// MaxBatchSize is the number of requests sent per $batch call. SharePoint
// throttles very large batches, so bigger inputs are split.
const MaxBatchSize = 100

// BatchRequest is one call inside a batch.
type BatchRequest struct {
	Method string
	// Path is relative to /_api, e.g. "web/lists/getbytitle('Tasks')/items".
	Path    string
	Headers map[string]string
	// Body is JSON encoded when not nil.
	Body any
}

// BatchResponse is the result of one call inside a batch.
type BatchResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

// Err returns an *httpx.Error when the sub-request did not succeed.
func (r BatchResponse) Err() error {
	if r.Status >= 200 && r.Status < 300 {
		return nil
	}
	return httpx.NewError(r.Status, r.Header, r.Body)
}

// JSON decodes the sub-response body into out; empty bodies are ignored.
func (r BatchResponse) JSON(out any) error {
	if out == nil || len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, out)
}

// Batch sends the requests through $batch, splitting them into groups of
// MaxBatchSize. Reads go in the batch directly; every write gets its own
// changeset, so one failing write does not roll back the others. Results are
// returned in input order. A failing sub-request does not fail the call: check
// BatchResponse.Err for each.
func (c *Client) Batch(ctx context.Context, reqs []BatchRequest) ([]BatchResponse, error) {
	out := make([]BatchResponse, 0, len(reqs))
	for start := 0; start < len(reqs); start += MaxBatchSize {
		chunk := reqs[start:min(start+MaxBatchSize, len(reqs))]
		boundary := "batch_" + newGUID()
		body, err := c.encodeBatch(boundary, chunk)
		if err != nil {
			return nil, err
		}
		req := httpx.NewRequest(http.MethodPost, "$batch").
			WithBytes(body, "multipart/mixed; boundary="+boundary).
			SetHeader("Accept", "multipart/mixed")
		resp, err := c.raw.Do(ctx, req)
		if err != nil {
			return nil, err
		}
		parts, err := decodeBatch(resp)
		if err != nil {
			return nil, err
		}
		if len(parts) != len(chunk) {
			return nil, fmt.Errorf("sharepoint: batch returned %d responses for %d requests", len(parts), len(chunk))
		}
		out = append(out, parts...)
	}
	return out, nil
}

func isRead(m string) bool { return m == "" || m == http.MethodGet || m == http.MethodHead }

func (c *Client) encodeBatch(boundary string, reqs []BatchRequest) ([]byte, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.SetBoundary(boundary); err != nil {
		return nil, err
	}
	for _, r := range reqs {
		method := r.Method
		if method == "" {
			method = http.MethodGet
		}
		var payload []byte
		if r.Body != nil {
			b, err := json.Marshal(r.Body)
			if err != nil {
				return nil, fmt.Errorf("sharepoint: marshal batch body: %w", err)
			}
			payload = b
		}
		var line bytes.Buffer
		fmt.Fprintf(&line, "%s %s/_api/%s HTTP/1.1\r\n", method, c.siteURL, strings.TrimPrefix(r.Path, "/"))
		line.WriteString("Accept: application/json;odata=nometadata\r\n")
		if payload != nil || !isRead(method) {
			ct := spx.JSONContentType
			if v, ok := r.Headers["Content-Type"]; ok {
				ct = v
			}
			fmt.Fprintf(&line, "Content-Type: %s\r\n", ct)
		}
		for k, v := range r.Headers {
			if !strings.EqualFold(k, "Content-Type") {
				fmt.Fprintf(&line, "%s: %s\r\n", k, v)
			}
		}
		line.WriteString("\r\n")
		line.Write(payload)

		appHTTP := textproto.MIMEHeader{"Content-Type": {"application/http"}, "Content-Transfer-Encoding": {"binary"}}
		if isRead(method) {
			w, err := mw.CreatePart(appHTTP)
			if err != nil {
				return nil, err
			}
			w.Write(line.Bytes())
			continue
		}
		cs := "changeset_" + newGUID()
		w, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {"multipart/mixed; boundary=" + cs}})
		if err != nil {
			return nil, err
		}
		inner := multipart.NewWriter(w)
		if err := inner.SetBoundary(cs); err != nil {
			return nil, err
		}
		pw, err := inner.CreatePart(appHTTP)
		if err != nil {
			return nil, err
		}
		pw.Write(line.Bytes())
		inner.Close()
	}
	mw.Close()
	return buf.Bytes(), nil
}

func decodeBatch(resp *httpx.Response) ([]BatchResponse, error) {
	mt, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mt, "multipart/") {
		return nil, fmt.Errorf("sharepoint: unexpected batch response type %q", resp.Header.Get("Content-Type"))
	}
	return readParts(bytes.NewReader(resp.Body), params["boundary"])
}

func readParts(r io.Reader, boundary string) ([]BatchResponse, error) {
	var out []BatchResponse
	mr := multipart.NewReader(r, boundary)
	for {
		p, err := mr.NextRawPart()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		mt, params, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
		if strings.HasPrefix(mt, "multipart/") {
			sub, err := readParts(p, params["boundary"])
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
			continue
		}
		hr, err := http.ReadResponse(bufio.NewReader(p), nil)
		if err != nil {
			return nil, fmt.Errorf("sharepoint: parse batch part: %w", err)
		}
		b, err := io.ReadAll(hr.Body)
		hr.Body.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, BatchResponse{Status: hr.StatusCode, Header: hr.Header, Body: b})
	}
}

func newGUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
