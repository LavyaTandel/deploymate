package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"deploymate/internal/config"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrNoToken      = errors.New("no token provided")
)

type Claims struct {
	jwt.RegisteredClaims
	OrgID     string   `json:"org_id"`
	ProjectID string   `json:"project_id"`
	Roles     []string `json:"roles"`
	Email     string   `json:"email"`
}

type OIDCValidator struct {
	issuer   string
	audience string
	jwks     *jwksCache
}

type jwksCache struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
	ttl     time.Duration
}

func NewOIDCValidator(cfg *config.Config) *OIDCValidator {
	return &OIDCValidator{
		issuer:   cfg.OIDC.Issuer,
		audience: cfg.OIDC.Audience,
		jwks: &jwksCache{
			keys: make(map[string]*rsa.PublicKey),
			ttl:  24 * time.Hour,
		},
	}
}

func (v *OIDCValidator) Validate(ctx context.Context, tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrNoToken
	}

	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		return nil, ErrNoToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, ErrInvalidToken
		}

		key, err := v.jwks.getKey(ctx, v.issuer, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Issuer != v.issuer {
		return nil, ErrInvalidToken
	}

	if !contains(claims.Audience, v.audience) {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (c *jwksCache) getKey(ctx context.Context, issuer, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if key, ok := c.keys[kid]; ok && time.Since(c.fetched) < c.ttl {
		c.mu.RUnlock()
		return key, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if key, ok := c.keys[kid]; ok && time.Since(c.fetched) < c.ttl {
		return key, nil
	}

	jwksURL := strings.TrimSuffix(issuer, "/") + "/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, "GET", jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			KID string `json:"kid"`
			KTY string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	c.keys = make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.KTY == "RSA" {
			n, _ := base64.RawURLEncoding.DecodeString(k.N)
			e, _ := base64.RawURLEncoding.DecodeString(k.E)
			pubKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes(n),
				E: int(new(big.Int).SetBytes(e).Int64()),
			}
			c.keys[k.KID] = pubKey
		}
	}
	c.fetched = time.Now()

	key, ok := c.keys[kid]
	if !ok {
		return nil, ErrInvalidToken
	}
	return key, nil
}

func contains(audience []string, target string) bool {
	for _, a := range audience {
		if a == target {
			return true
		}
	}
	return false
}

type contextKey string

const (
	ClaimsKey contextKey = "claims"
	OrgIDKey  contextKey = "org_id"
)

func ClaimsFromContext(ctx context.Context) *Claims {
	if c, ok := ctx.Value(ClaimsKey).(*Claims); ok {
		return c
	}
	return nil
}

func OrgIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(OrgIDKey).(string); ok {
		return id
	}
	return ""
}

func RequireOIDC(validator *OIDCValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			claims, err := validator.Validate(r.Context(), authHeader)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			ctx = context.WithValue(ctx, OrgIDKey, claims.OrgID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
