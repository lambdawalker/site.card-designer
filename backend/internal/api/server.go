package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/lambdawalker/site.card-designer/backend/internal/catalog"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const MaxBundle = 4 << 20

var ErrLimit = errors.New("size limit exceeded")
var safeID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)

func fmtInt(n int64) string { return strconv.FormatInt(n, 10) }
func readLimit(r io.Reader, n int64) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, n+1))
	if int64(len(b)) > n {
		return nil, ErrLimit
	}
	return b, e
}
func randomID() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}

type Config struct {
	CognitoDomain string `json:"cognito_domain"`
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	LogoutURI     string `json:"logout_uri"`
	Region        string `json:"region"`
	UserPoolID    string `json:"user_pool_id"`
}
type Bundle struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Layouts []json.RawMessage `json:"layouts"`
}
type Server struct {
	Config  Config
	Auth    Authenticator
	Objects Objects
	Tokens  Tokens
	HF      *HF
	Catalog []byte
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]string{"error": msg})
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Card-Designer-API-Version", "1")
	p := strings.TrimSuffix(r.URL.Path, "/")
	if p == "/api/config" && r.Method == "GET" {
		reply(w, 200, s.Config)
		return
	}
	if (p == "/health" || p == "/api/health") && r.Method == "GET" {
		reply(w, 200, map[string]string{"status": "ok"})
		return
	}
	if !strings.HasPrefix(p, "/api/") {
		fail(w, 404, "not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		fail(w, 401, "authentication required")
		return
	}
	id, e := s.Auth.Authenticate(r.Context(), strings.TrimPrefix(auth, "Bearer "))
	if e != nil {
		fail(w, 401, "invalid credentials")
		return
	}
	if id.ReadOnly && (r.Method != "GET" || (p == "/api/tokens" || strings.HasPrefix(p, "/api/tokens/"))) {
		fail(w, 403, "read-only token")
		return
	}
	p = strings.Replace(p, "/api/v1/bundles", "/api/bundles", 1)
	if p == "/api/templates" || strings.HasPrefix(p, "/api/templates/") {
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return
		}
		b := s.Catalog
		if len(b) == 0 {
			b = catalog.Data
		}
		if p == "/api/templates" {
			var v any
			if json.Unmarshal(b, &v) != nil {
				fail(w, 500, "catalog unavailable")
				return
			}
			reply(w, 200, v)
			return
		}
		var rows []map[string]any
		json.Unmarshal(b, &rows)
		for _, v := range rows {
			if v["id"] == strings.TrimPrefix(p, "/api/templates/") {
				reply(w, 200, v)
				return
			}
		}
		fail(w, 404, "template not found")
		return
	}
	if p == "/api/bundles" || strings.HasPrefix(p, "/api/bundles/") {
		s.bundles(w, r, p, id)
		return
	}
	if p == "/api/tokens" || strings.HasPrefix(p, "/api/tokens/") {
		s.tokens(w, r, p, id)
		return
	}
	if strings.HasPrefix(p, "/api/datasets") {
		s.datasets(w, r, p)
		return
	}
	fail(w, 404, "not found")
}
func prefix(owner string) string { return "users/" + hash(owner) + "/bundles/" }
func decodeBundle(r *http.Request) (Bundle, error) {
	var b Bundle
	raw, e := readLimit(r.Body, MaxBundle)
	if e != nil {
		return b, e
	}
	if e = json.Unmarshal(raw, &b); e != nil {
		return b, e
	}
	if strings.TrimSpace(b.Name) == "" || len(b.Name) > 200 || len(b.Layouts) == 0 || len(b.Layouts) > 100 {
		return b, errors.New("invalid bundle")
	}
	seen := map[string]bool{}
	for i, l := range b.Layouts {
		var obj map[string]any
		if json.Unmarshal(l, &obj) != nil || obj == nil {
			return b, errors.New("invalid layout")
		}
		lid, _ := obj["layout_id"].(string)
		if lid == "" {
			lid = randomID()
			obj["layout_id"] = lid
			b.Layouts[i], _ = json.Marshal(obj)
		}
		if !safeID.MatchString(lid) || seen[lid] {
			return b, errors.New("invalid layout_id")
		}
		seen[lid] = true
	}
	return b, nil
}
func (s *Server) bundles(w http.ResponseWriter, r *http.Request, p string, id Identity) {
	rest := strings.TrimPrefix(p, "/api/bundles")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	base := prefix(id.Owner)
	if rest == "" {
		switch r.Method {
		case "GET":
			keys, e := s.Objects.List(r.Context(), base)
			if e != nil {
				fail(w, 503, "storage unavailable")
				return
			}
			rows := []map[string]any{}
			for _, k := range keys {
				raw, e := s.Objects.Get(r.Context(), k, MaxBundle)
				if e != nil {
					fail(w, 503, "storage unavailable")
					return
				}
				var b Bundle
				if json.Unmarshal(raw, &b) != nil {
					continue
				}
				rows = append(rows, map[string]any{"id": b.ID, "name": b.Name, "layout_count": len(b.Layouts)})
			}
			reply(w, 200, rows)
		case "POST":
			b, e := decodeBundle(r)
			if e != nil {
				if errors.Is(e, ErrLimit) {
					fail(w, 413, "bundle exceeds 4 MiB limit")
				} else {
					fail(w, 400, "invalid bundle")
				}
				return
			}
			b.ID = randomID()
			raw, _ := json.Marshal(b)
			if len(raw) > MaxBundle {
				fail(w, 413, "bundle exceeds 4 MiB limit")
				return
			}
			if s.Objects.Put(r.Context(), base+b.ID+".json", raw) != nil {
				fail(w, 503, "storage unavailable")
				return
			}
			reply(w, 201, b)
		default:
			fail(w, 405, "method not allowed")
		}
		return
	}
	if len(parts) > 2 || !safeID.MatchString(parts[0]) {
		fail(w, 400, "invalid id")
		return
	}
	key := base + parts[0] + ".json"
	if len(parts) == 2 {
		if parts[1] != "clone" || r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return
		}
		raw, e := s.Objects.Get(r.Context(), key, MaxBundle)
		var b Bundle
		if e != nil || json.Unmarshal(raw, &b) != nil {
			fail(w, 404, "bundle not found")
			return
		}
		draft, e := readLimit(r.Body, MaxBundle)
		if e != nil {
			fail(w, 413, "bundle exceeds 4 MiB limit")
			return
		}
		if len(bytes.TrimSpace(draft)) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(draft))
			b, e = decodeBundle(r)
			if e != nil {
				fail(w, 400, "invalid bundle draft")
				return
			}
		}
		b.ID = randomID()
		b.Name += " (copy)"
		raw, _ = json.Marshal(b)
		if len(raw) > MaxBundle {
			fail(w, 413, "bundle exceeds 4 MiB limit")
			return
		}
		if s.Objects.Put(r.Context(), base+b.ID+".json", raw) != nil {
			fail(w, 503, "storage unavailable")
			return
		}
		reply(w, 201, b)
		return
	}
	switch r.Method {
	case "GET":
		raw, e := s.Objects.Get(r.Context(), key, MaxBundle)
		if e != nil {
			if errors.Is(e, ErrMissing) {
				fail(w, 404, "bundle not found")
			} else {
				fail(w, 503, "storage unavailable")
			}
			return
		}
		var b Bundle
		if json.Unmarshal(raw, &b) != nil {
			fail(w, 500, "invalid stored bundle")
			return
		}
		reply(w, 200, b)
	case "PUT":
		if _, e := s.Objects.Get(r.Context(), key, MaxBundle); e != nil {
			if errors.Is(e, ErrMissing) {
				fail(w, 404, "bundle not found")
			} else {
				fail(w, 503, "storage unavailable")
			}
			return
		}
		b, e := decodeBundle(r)
		if e != nil {
			if errors.Is(e, ErrLimit) {
				fail(w, 413, "bundle exceeds 4 MiB limit")
			} else {
				fail(w, 400, "invalid bundle")
			}
			return
		}
		b.ID = parts[0]
		raw, _ := json.Marshal(b)
		if len(raw) > MaxBundle {
			fail(w, 413, "bundle exceeds 4 MiB limit")
			return
		}
		if s.Objects.Put(r.Context(), key, raw) != nil {
			fail(w, 503, "storage unavailable")
			return
		}
		reply(w, 200, b)
	case "DELETE":
		if s.Objects.Delete(r.Context(), key) != nil {
			fail(w, 503, "storage unavailable")
			return
		}
		w.WriteHeader(204)
	default:
		fail(w, 405, "method not allowed")
	}
}
func (s *Server) tokens(w http.ResponseWriter, r *http.Request, p string, id Identity) {
	if p == "/api/tokens" {
		switch r.Method {
		case "GET":
			ts, e := s.Tokens.List(r.Context(), id.Owner)
			if e != nil {
				fail(w, 503, "token storage unavailable")
				return
			}
			out := []Token{}
			for _, t := range ts {
				if t.Expires > time.Now().Unix() {
					out = append(out, t)
				}
			}
			reply(w, 200, out)
		case "POST":
			raw, e := readLimit(r.Body, 4096)
			var body struct {
				Name string `json:"name"`
			}
			if e != nil || json.Unmarshal(raw, &body) != nil || strings.TrimSpace(body.Name) == "" || len(body.Name) > 100 {
				fail(w, 400, "invalid name")
				return
			}
			plain := "cd_" + randomID()
			t := Token{hash(plain), id.Owner, body.Name, time.Now().Add(90 * 24 * time.Hour).Unix()}
			if s.Tokens.Put(r.Context(), t) != nil {
				fail(w, 503, "token storage unavailable")
				return
			}
			reply(w, 201, map[string]any{"id": t.ID, "name": t.Name, "expires_at": t.Expires, "token": plain})
		default:
			fail(w, 405, "method not allowed")
		}
		return
	}
	tid := strings.TrimPrefix(p, "/api/tokens/")
	if r.Method != "DELETE" || !safeID.MatchString(tid) {
		fail(w, 405, "method not allowed")
		return
	}
	t, e := s.Tokens.Get(r.Context(), tid)
	if e != nil || t.Owner != id.Owner {
		fail(w, 404, "token not found")
		return
	}
	if s.Tokens.Delete(r.Context(), tid) != nil {
		fail(w, 503, "token storage unavailable")
		return
	}
	w.WriteHeader(204)
}
