package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type Server struct {
	e      *echo.Echo
	priv   *rsa.PrivateKey
	jwks   jwk.Set
	keyID  string
	issuer string
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func New() (*Server, error) {
	e := echo.New()

	// 1) 秘密鍵ロード（AUTH_PRIVATE_KEY_FILE があれば利用、無ければ生成）
	priv, err := loadOrGenerateKey(getenv("AUTH_PRIVATE_KEY_FILE", ""))
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}

	// 2) 公開鍵をJWK化（kid 自動付与）→ JWKSに載せる
	pubJWK, err := jwk.FromRaw(priv.Public())
	if err != nil {
		return nil, fmt.Errorf("jwk from public: %w", err)
	}
	if err := jwk.AssignKeyID(pubJWK); err != nil {
		return nil, fmt.Errorf("assign kid: %w", err)
	}
	keyID := pubJWK.KeyID()
	set := jwk.NewSet()
	set.AddKey(pubJWK)

	s := &Server{
		e:      e,
		priv:   priv,
		jwks:   set,
		keyID:  keyID,
		issuer: getenv("AUTH_ISSUER", "http://localhost:18080"),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	// Liveness
	s.e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// JWKS（公開鍵配布）
	s.e.GET("/.well-known/jwks.json", func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "application/json")
		return json.NewEncoder(c.Response()).Encode(s.jwks)
	})

	// ダミーログイン：email が入っていればOK、RS256で署名したJWTを返す
	s.e.POST("/auth/login", func(c echo.Context) error {
		var req loginReq
		if err := c.Bind(&req); err != nil || req.Email == "" {
			return c.JSON(http.StatusUnprocessableEntity, map[string]any{"error": "invalid payload"})
		}

		now := time.Now()
		claims := jwt.MapClaims{
			"sub":   req.Email,
			"iss":   s.issuer, // ← Kong 側の jwt_secrets.key と一致させる
			"aud":   "vote-app",
			"iat":   now.Unix(),
			"exp":   now.Add(30 * time.Minute).Unix(),
			"scope": "read write",
		}

		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = s.keyID
		signed, err := tok.SignedString(s.priv)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "sign failed"})
		}

		return c.JSON(http.StatusOK, map[string]any{
			"access_token": signed,
			"token_type":   "Bearer",
			"expires_in":   1800,
		})
	})
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

// ===== helpers =====

func loadOrGenerateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		// dev: 未指定なら都度生成（※Kong検証する場合は固定鍵を使ってください）
		return rsa.GenerateKey(rand.Reader, 2048)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pem: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("pem decode failed")
	}

	switch block.Type {
	case "RSA PRIVATE KEY": // PKCS#1
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse pkcs1: %w", err)
		}
		return key, nil
	case "PRIVATE KEY": // PKCS#8
		any, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse pkcs8: %w", err)
		}
		rsaKey, ok := any.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported pem type: %s", block.Type)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
