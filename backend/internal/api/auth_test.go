package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWTSignatureClaimsAndJWKSCache(t *testing.T) {
	k, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	hits := 0
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "k1", "n": base64.RawURLEncoding.EncodeToString(k.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes())}}})
	}))
	defer up.Close()
	a := &Auth{Issuer: up.URL, ClientID: "client", HTTP: up.Client()}
	sign := func(c jwt.MapClaims, key *rsa.PrivateKey) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
		tok.Header["kid"] = "k1"
		s, e := tok.SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	base := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": "alice", "iss": up.URL, "client_id": "client", "token_use": "access", "exp": time.Now().Add(time.Hour).Unix()}
	}
	for i := 0; i < 2; i++ {
		if _, e := a.Authenticate(context.Background(), sign(base(), k)); e != nil {
			t.Fatal(e)
		}
	}
	if hits != 1 {
		t.Fatal("jwks not cached")
	}
	for _, field := range []string{"iss", "client_id", "token_use", "exp"} {
		c := base()
		c[field] = "bad"
		if _, e := a.Authenticate(context.Background(), sign(c, k)); e == nil {
			t.Fatal("accepted wrong", field)
		}
	}
	c := base()
	delete(c, "exp")
	if _, e := a.Authenticate(context.Background(), sign(c, k)); e == nil {
		t.Fatal("missing exp")
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	if _, e := a.Authenticate(context.Background(), sign(base(), other)); e == nil {
		t.Fatal("invalid signature accepted")
	}
}
