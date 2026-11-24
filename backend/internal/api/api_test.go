package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type mem struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (m *mem) Get(_ context.Context, k string, _ int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.m[k]
	if !ok {
		return nil, ErrMissing
	}
	return v, nil
}
func (m *mem) Put(_ context.Context, k string, v []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[k] = v
	return nil
}
func (m *mem) Delete(_ context.Context, k string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
	return nil
}
func (m *mem) List(_ context.Context, p string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []string{}
	for k := range m.m {
		if strings.HasPrefix(k, p) {
			out = append(out, k)
		}
	}
	return out, nil
}

type tokens map[string]Token

func (t tokens) Get(_ context.Context, k string) (Token, error) {
	v, ok := t[k]
	if !ok {
		return v, ErrMissing
	}
	return v, nil
}
func (t tokens) Put(_ context.Context, v Token) error     { t[v.ID] = v; return nil }
func (t tokens) Delete(_ context.Context, k string) error { delete(t, k); return nil }
func (t tokens) List(_ context.Context, o string) ([]Token, error) {
	out := []Token{}
	for _, v := range t {
		if v.Owner == o {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakeAuth struct{}

func (fakeAuth) Authenticate(_ context.Context, v string) (Identity, error) {
	if v == "bad" {
		return Identity{}, errors.New("bad")
	}
	return Identity{v, v == "reader"}, nil
}
func call(s http.Handler, method, path, owner, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if owner != "" {
		r.Header.Set("Authorization", "Bearer "+owner)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestOwnershipAndReadOnly(t *testing.T) {
	s := &Server{Objects: &mem{m: map[string][]byte{}}, Tokens: tokens{}, Auth: fakeAuth{}}
	w := call(s, "POST", "/api/bundles", "alice", `{"name":"a","layouts":[{"template":"ok"}]}`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	var b Bundle
	json.Unmarshal(w.Body.Bytes(), &b)
	if call(s, "GET", "/api/v1/bundles/"+b.ID, "bob", "").Code != 404 {
		t.Fatal("cross-owner read")
	}
	call(s, "DELETE", "/api/bundles/"+b.ID, "bob", "")
	if call(s, "GET", "/api/bundles/"+b.ID, "alice", "").Code != 200 {
		t.Fatal("other owner deleted")
	}
	for _, p := range []string{"/api/bundles", "/api/tokens", "/api/bundles/x/clone"} {
		if call(s, "POST", p, "reader", `{}`).Code != 403 {
			t.Fatal("read token wrote", p)
		}
	}
	if call(s, "GET", "/api/tokens", "reader", "").Code != 403 {
		t.Fatal("read token listed tokens")
	}
	if call(s, "GET", "/api/templates", "", "").Code != 401 {
		t.Fatal("anonymous templates")
	}
}
func TestTokenHashExpiryRevocation(t *testing.T) {
	ts := tokens{}
	s := &Server{Tokens: ts, Auth: fakeAuth{}}
	w := call(s, "POST", "/api/tokens", "alice", `{"name":"renderer"}`)
	var issued struct{ ID, Token string }
	json.Unmarshal(w.Body.Bytes(), &issued)
	if len(issued.Token) != 67 || issued.ID != hash(issued.Token) {
		t.Fatal(w.Body)
	}
	a := &Auth{Tokens: ts}
	if id, e := a.Authenticate(context.Background(), issued.Token); e != nil || !id.ReadOnly || id.Owner != "alice" {
		t.Fatal(id, e)
	}
	if call(s, "DELETE", "/api/tokens/"+issued.ID, "bob", "").Code != 404 {
		t.Fatal("other user revoked")
	}
	call(s, "DELETE", "/api/tokens/"+issued.ID, "alice", "")
	if _, e := a.Authenticate(context.Background(), issued.Token); e == nil {
		t.Fatal("revoked accepted")
	}
	ts[issued.ID] = Token{issued.ID, "alice", "x", time.Now().Add(-time.Second).Unix()}
	if _, e := a.Authenticate(context.Background(), issued.Token); e == nil {
		t.Fatal("expired accepted")
	}
}
func TestLiveHF(t *testing.T) {
	if os.Getenv("LIVE_HF") != "1" {
		t.Skip("LIVE_HF=1 public upstream smoke")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	h := &HF{Objects: &mem{m: map[string][]byte{}}}
	l, e := h.Packages(ctx)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("revision", l.Revision, "packages", len(l.Packages))
	raw, e := h.Package(ctx, l.Revision, l.Packages[0].Path)
	if e != nil {
		t.Fatal(e)
	}
	var out struct{ Rows []map[string]any }
	json.Unmarshal(raw, &out)
	if len(out.Rows) == 0 {
		t.Fatal("empty")
	}
	image, _ := out.Rows[0]["image"].(map[string]any)
	if _, ok := image["bytes"].(string); !ok {
		t.Fatal("image bytes not base64", image)
	}
	t.Log("rows", len(out.Rows), "response bytes", len(raw))
}
func TestPathValidationAndDiscoveryCache(t *testing.T) {
	hits := 0
	rev := strings.Repeat("a", 40)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if strings.Contains(r.URL.Path, "revision/main") {
			json.NewEncoder(w).Encode(map[string]string{"sha": rev})
		} else {
			json.NewEncoder(w).Encode([]map[string]any{{"type": "file", "path": "data/train/test.parquet", "size": 50}})
		}
	}))
	defer up.Close()
	m := &mem{m: map[string][]byte{}}
	h := &HF{Objects: m, BaseURL: up.URL, HTTP: up.Client()}
	for i := 0; i < 2; i++ {
		if _, e := h.Packages(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	if hits != 2 {
		t.Fatal("discovery cache missed", hits)
	}
	for _, path := range []string{"../x.parquet", "data/train/../x.parquet", "https://evil/x.parquet", "data/train/x.parquet?foo"} {
		if _, e := h.Package(context.Background(), rev, path); e == nil {
			t.Fatal("unsafe path", path)
		}
	}
	key := "cache/hf/photo-id/" + rev + "/data/train/test.json"
	m.Put(context.Background(), key, []byte(`{"revision":"cached","rows":[]}`))
	for i := 0; i < 2; i++ {
		if _, e := h.Package(context.Background(), rev, "data/train/test.parquet"); e != nil {
			t.Fatal(e)
		}
	}
	if hits != 2 {
		t.Fatal("S3 cache repeated upstream")
	}
}
func TestDraftCloneAndExistingPut(t *testing.T) {
	s := &Server{Objects: &mem{m: map[string][]byte{}}, Tokens: tokens{}, Auth: fakeAuth{}}
	w := call(s, "POST", "/api/bundles", "alice", `{"name":"original","layouts":[{"layout_id":"l1","template":"old"}]}`)
	var b Bundle
	json.Unmarshal(w.Body.Bytes(), &b)
	draft := `{"name":"draft","layouts":[{"layout_id":"l1","template":"edited","sample_source":{"revision":"abc"}}]}`
	w = call(s, "POST", "/api/bundles/"+b.ID+"/clone", "alice", draft)
	if w.Code != 201 || !strings.Contains(w.Body.String(), "edited") || !strings.Contains(w.Body.String(), "sample_source") {
		t.Fatal(w.Code, w.Body)
	}
	if call(s, "PUT", "/api/bundles/missing", "alice", draft).Code != 404 {
		t.Fatal("PUT created missing bundle")
	}
	if call(s, "POST", "/api/bundles", "alice", strings.Repeat("x", MaxBundle+1)).Code != 413 {
		t.Fatal("oversize not rejected clearly")
	}
	if call(s, "POST", "/api/bundles", "alice", `{"name":"x","layouts":[{"layout_id":"x"},{"layout_id":"x"}]}`).Code != 400 {
		t.Fatal("duplicate layout id accepted")
	}
}
