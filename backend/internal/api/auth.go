package api

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Identity struct {
	Owner    string
	ReadOnly bool
}
type Authenticator interface {
	Authenticate(context.Context, string) (Identity, error)
}
type Auth struct {
	Issuer, ClientID string
	Tokens           Tokens
	HTTP             *http.Client
	mu               sync.Mutex
	keys             map[string]*rsa.PublicKey
	fetched          time.Time
}

func hash(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }
func (a *Auth) Authenticate(ctx context.Context, s string) (Identity, error) {
	if len(s) > 16384 {
		return Identity{}, errors.New("invalid token")
	}
	if strings.HasPrefix(s, "cd_") {
		if len(s) != 67 {
			return Identity{}, errors.New("invalid token")
		}
		t, e := a.Tokens.Get(ctx, hash(s))
		if e != nil || t.Expires <= time.Now().Unix() || t.Owner == "" {
			return Identity{}, errors.New("invalid token")
		}
		return Identity{t.Owner, true}, nil
	}
	claims := jwt.MapClaims{}
	t, e := jwt.ParseWithClaims(s, claims, func(t *jwt.Token) (any, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok || len(kid) > 200 {
			return nil, errors.New("invalid key")
		}
		return a.key(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(a.Issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || !t.Valid || claims["token_use"] != "access" || claims["client_id"] != a.ClientID {
		return Identity{}, errors.New("invalid token")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" || len(sub) > 200 {
		return Identity{}, errors.New("invalid subject")
	}
	return Identity{sub, false}, nil
}
func (a *Auth) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if time.Since(a.fetched) < time.Hour {
		if k := a.keys[kid]; k != nil {
			return k, nil
		}
		if time.Since(a.fetched) < time.Minute {
			return nil, errors.New("unknown key")
		}
	}
	req, e := http.NewRequestWithContext(ctx, "GET", a.Issuer+"/.well-known/jwks.json", nil)
	if e != nil {
		return nil, e
	}
	client := a.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	r, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, errors.New("key discovery unavailable")
	}
	b, e := readLimit(r.Body, 1<<20)
	if e != nil {
		return nil, e
	}
	var doc struct {
		Keys []struct{ Kid, Kty, Use, Alg, N, E string }
	}
	if json.Unmarshal(b, &doc) != nil || len(doc.Keys) > 10 {
		return nil, errors.New("invalid jwks")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, j := range doc.Keys {
		if j.Kty != "RSA" || j.Alg != "RS256" || j.Use != "sig" {
			continue
		}
		n, e1 := base64.RawURLEncoding.DecodeString(j.N)
		eb, e2 := base64.RawURLEncoding.DecodeString(j.E)
		if e1 != nil || e2 != nil || len(n) < 256 || len(n) > 1024 || len(eb) > 4 {
			continue
		}
		exp := 0
		for _, v := range eb {
			exp = exp<<8 + int(v)
		}
		if exp < 3 {
			continue
		}
		keys[j.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exp}
	}
	a.keys = keys
	a.fetched = time.Now()
	if keys[kid] == nil {
		return nil, errors.New("unknown key")
	}
	return keys[kid], nil
}
