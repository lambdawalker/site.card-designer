package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVaultScopeAndTamper(t *testing.T) {
	v, e := encrypt("dev", []byte("password"), []byte("secret"))
	if e != nil {
		t.Fatal(e)
	}
	b, e := decrypt("dev", []byte("password"), v)
	if e != nil || !bytes.Equal(b, []byte("secret")) {
		t.Fatal(e)
	}
	if _, e = decrypt("prod", []byte("password"), v); e == nil {
		t.Fatal("scope accepted")
	}
	if _, e = decrypt("dev", []byte("wrong"), v); e == nil {
		t.Fatal("password accepted")
	}
	v.Data[0] ^= 1
	if _, e = decrypt("dev", []byte("password"), v); e == nil {
		t.Fatal("tamper accepted")
	}
}
func TestSettingsPrecedence(t *testing.T) {
	s := Settings{Region: "eu-west-1"}
	merge(&s, Settings{Region: "us-east-1", State: "s3://bucket/one", Initialized: true})
	if s.Region != "eu-west-1" || !s.Initialized || s.State == "" {
		t.Fatal(s)
	}
}
func TestAtomicPrivate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	if e := save(p, Settings{Project: "one"}); e != nil {
		t.Fatal(e)
	}
	st, e := os.Stat(p)
	if e != nil || st.Mode().Perm() != 0600 {
		t.Fatal(st, e)
	}
}
func TestTLSPolicy(t *testing.T) {
	valid := `{"Statement":[{"Effect":"Deny","Principal":"*","Action":"s3:*","Resource":["arn:aws:s3:::state","arn:aws:s3:::state/*"],"Condition":{"Bool":{"aws:SecureTransport":"false"}}}]}`
	if !tlsPolicy(valid, "state") {
		t.Fatal("valid policy rejected")
	}
	for _, bad := range []string{`{"Statement":[]}`, string(bytes.ReplaceAll([]byte(valid), []byte(`"Deny"`), []byte(`"Allow"`))), string(bytes.ReplaceAll([]byte(valid), []byte(`"false"`), []byte(`"true"`)))} {
		if tlsPolicy(bad, "state") {
			t.Fatal("unsafe policy accepted")
		}
	}
}
func TestBuildArtifactsTamper(t *testing.T) {
	old := root
	root = t.TempDir()
	defer func() { root = old }()
	atomic(filepath.Join(root, "backend", "bootstrap.zip"), []byte("zip"))
	atomic(filepath.Join(root, "web", "dist", "index.html"), []byte("web"))
	a, e := buildArtifacts()
	if e != nil {
		t.Fatal(e)
	}
	atomic(filepath.Join(root, "web", "dist", "index.html"), []byte("changed"))
	b, e := buildArtifacts()
	if e != nil || a == b {
		t.Fatal("tamper missed", e)
	}
}

func TestCountdownOnlyAcceptsSilence(t *testing.T) {
	d := time.Unix(100, 0)
	for _, c := range []struct {
		timed bool
		n     int
		now   time.Time
		want  bool
	}{{true, 0, d.Add(-time.Second), false}, {true, 0, d, true}, {true, 1, d, false}, {false, 0, d, false}, {false, 1, d.Add(time.Hour), false}} {
		if got := timeoutAcceptsDefault(c.timed, c.n, c.now, d); got != c.want {
			t.Fatalf("%+v got %v", c, got)
		}
	}
}

func TestAWSIdentityLengths(t *testing.T) {
	if validateIdentity("card-designer", "dev") != nil {
		t.Fatal("valid identity rejected")
	}
	if validateIdentity(strings.Repeat("a", 36), "dev") == nil {
		t.Fatal("oversized state bucket name accepted")
	}
	if validateIdentity(strings.Repeat("a", 35), strings.Repeat("b", 15)) == nil {
		t.Fatal("oversized Cognito domain accepted")
	}
}
