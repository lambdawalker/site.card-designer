package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/parquet-go/parquet-go"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const dataset = "lambdaWalker/ds.photo-id"
const maxDownload = 24 << 20
const maxResponse = 5 << 20

var revisionRE = regexp.MustCompile(`^[a-f0-9]{40}$`)
var packageRE = regexp.MustCompile(`^data/train/[A-Za-z0-9_-]+\.parquet$`)

type PackageInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}
type PackageList struct {
	Revision string        `json:"revision"`
	Packages []PackageInfo `json:"packages"`
}
type HF struct {
	Objects  Objects
	Token    string
	HTTP     *http.Client
	BaseURL  string
	mu       sync.Mutex
	cached   PackageList
	until    time.Time
	gateOnce sync.Once
	gate     chan struct{}
}

func (h *HF) fetch(ctx context.Context, path string, limit int64, nextPage ...*string) ([]byte, error) {
	base := h.BaseURL
	if base == "" {
		base = "https://huggingface.co"
	}
	client := h.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 4 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return errors.New("unsafe redirect")
			}
			req.Header.Del("Authorization")
			return nil
		}}
	}
	for attempt := 0; attempt < 3; attempt++ {
		delay := time.Duration(attempt+1) * time.Second
		req, e := http.NewRequestWithContext(ctx, "GET", base+path, nil)
		if e != nil {
			return nil, e
		}
		if h.Token != "" {
			req.Header.Set("Authorization", "Bearer "+h.Token)
		}
		r, e := client.Do(req)
		if e == nil {
			if r.StatusCode == 200 {
				if len(nextPage) > 0 {
					link := r.Header.Get("Link")
					for _, part := range strings.Split(link, ",") {
						if strings.Contains(part, `rel="next"`) {
							l := strings.Index(part, "<")
							rr := strings.Index(part, ">")
							if l >= 0 && rr > l {
								u, err := url.Parse(part[l+1 : rr])
								bu, _ := url.Parse(base)
								if err != nil || u.Host != bu.Host || u.Scheme != bu.Scheme || !strings.HasPrefix(u.Path, "/api/datasets/"+dataset+"/tree/") {
									r.Body.Close()
									return nil, errors.New("invalid dataset pagination")
								}
								*nextPage[0] = u.RequestURI()
							}
						}
					}
				}
				b, e := readLimit(r.Body, limit)
				r.Body.Close()
				return b, e
			}
			if seconds, err := strconv.Atoi(r.Header.Get("Retry-After")); err == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
				if delay > 5*time.Second {
					delay = 5 * time.Second
				}
			}
			io.Copy(io.Discard, io.LimitReader(r.Body, 4096))
			r.Body.Close()
			if r.StatusCode != 429 && r.StatusCode < 500 {
				return nil, fmt.Errorf("dataset upstream status %d", r.StatusCode)
			}
		}
		if attempt == 2 {
			return nil, errors.New("dataset upstream unavailable")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, errors.New("dataset upstream unavailable")
}
func (h *HF) Packages(ctx context.Context) (PackageList, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if time.Now().Before(h.until) {
		return h.cached, nil
	}
	raw, e := h.fetch(ctx, "/api/datasets/"+dataset+"/revision/main", 1<<20)
	if e != nil {
		return PackageList{}, e
	}
	var meta struct {
		SHA string `json:"sha"`
	}
	if json.Unmarshal(raw, &meta) != nil || !revisionRE.MatchString(meta.SHA) {
		return PackageList{}, errors.New("invalid dataset revision")
	}
	out := PackageList{meta.SHA, []PackageInfo{}}
	next := "/api/datasets/" + dataset + "/tree/" + meta.SHA + "/data/train?limit=1000"
	for page := 0; next != ""; page++ {
		if page >= 10 {
			return PackageList{}, errors.New("dataset package discovery exceeds 10000 entries")
		}
		path := next
		next = ""
		raw, e = h.fetch(ctx, path, 2<<20, &next)
		if e != nil {
			return PackageList{}, e
		}
		var tree []struct {
			Path, Type string
			Size       int64
		}
		if json.Unmarshal(raw, &tree) != nil || len(tree) > 1000 {
			return PackageList{}, errors.New("invalid dataset package list")
		}
		for _, v := range tree {
			if v.Type == "file" && packageRE.MatchString(v.Path) && v.Size > 0 && v.Size <= maxDownload {
				out.Packages = append(out.Packages, PackageInfo{v.Path, v.Size})
			}
		}
	}
	if len(out.Packages) == 0 {
		return PackageList{}, errors.New("no supported packages")
	}
	h.cached = out
	h.until = time.Now().Add(5 * time.Minute)
	return out, nil
}
func (h *HF) Package(ctx context.Context, rev, path string) (json.RawMessage, error) {
	h.gateOnce.Do(func() { h.gate = make(chan struct{}, 1) })
	select {
	case h.gate <- struct{}{}:
		defer func() { <-h.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if !revisionRE.MatchString(rev) || !packageRE.MatchString(path) {
		return nil, errors.New("invalid package path or revision")
	}
	key := "cache/hf/photo-id/" + rev + "/" + strings.TrimSuffix(path, ".parquet") + ".json"
	if raw, e := h.Objects.Get(ctx, key, maxResponse); e == nil {
		if !json.Valid(raw) {
			return nil, errors.New("invalid cached package")
		}
		return raw, nil
	} else if !errors.Is(e, ErrMissing) {
		return nil, errors.New("package cache unavailable")
	}
	raw, e := h.fetch(ctx, "/datasets/"+dataset+"/resolve/"+rev+"/"+"data/train/"+url.PathEscape(strings.TrimPrefix(path, "data/train/")), maxDownload)
	if e != nil {
		return nil, e
	}
	f, e := parquet.OpenFile(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		return nil, errors.New("invalid parquet")
	}
	if f.NumRows() > 500 || f.NumRows() < 0 {
		return nil, errors.New("package row limit exceeded")
	}
	rows := []map[string]any{}
	reader := parquet.NewReader(bytes.NewReader(raw))
	defer reader.Close()
	for i := int64(0); i < f.NumRows(); i++ {
		row := map[string]any{}
		if e := reader.Read(&row); e != nil {
			return nil, errors.New("unsupported parquet schema")
		}
		clean := map[string]any{}
		for _, k := range []string{"id", "prompt", "parameters", "image", "image_transparent"} {
			if v, ok := row[k]; ok {
				clean[k] = v
			}
		}
		rows = append(rows, clean)
		if i%10 == 0 {
			b, _ := json.Marshal(rows)
			if len(b) > maxResponse {
				return nil, errors.New("package response limit exceeded")
			}
		}
	}
	out, e := json.Marshal(map[string]any{"revision": rev, "path": path, "rows": rows})
	if e != nil || len(out) > maxResponse {
		return nil, errors.New("package response limit exceeded")
	}
	if e = h.Objects.Put(ctx, key, out); e != nil {
		return nil, errors.New("package cache unavailable")
	}
	return out, nil
}
func (s *Server) datasets(w http.ResponseWriter, r *http.Request, p string) {
	if r.Method != "GET" {
		fail(w, 405, "method not allowed")
		return
	}
	if p == "/api/datasets" {
		reply(w, 200, []map[string]any{{"id": "photo-id", "dataset": dataset, "name": "Synthetic portraits", "count": 0, "reference": dataset}})
		return
	}
	if s.HF == nil {
		fail(w, 503, "dataset service unavailable")
		return
	}
	if p == "/api/datasets/photo-id/packages" {
		v, e := s.HF.Packages(r.Context())
		if e != nil {
			fail(w, 502, e.Error())
			return
		}
		reply(w, 200, v)
		return
	}
	if p == "/api/datasets/photo-id/package" {
		v, e := s.HF.Package(r.Context(), r.URL.Query().Get("revision"), r.URL.Query().Get("path"))
		if e != nil {
			fail(w, 400, e.Error())
			return
		}
		reply(w, 200, v)
		return
	}
	fail(w, 404, "dataset not found")
}
