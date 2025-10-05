package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
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

	// dev用：起動時にRSA鍵を生成（本番はSecretから読み込み）
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	// JWK化
	k, _ := jwk.FromRaw(priv.Public())
	_ = jwk.AssignKeyID(k) // kid を自動付与
	keyID := k.KeyID()
	set := jwk.NewSet()
	set.AddKey(k)

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
	// ヘルスチェック
	s.e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// JWKS公開
	s.e.GET("/.well-known/jwks.json", func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "application/json")
		return json.NewEncoder(c.Response()).Encode(s.jwks)
	})

	// ダミーログイン
	s.e.POST("/auth/login", func(c echo.Context) error {
		var req loginReq
		if err := c.Bind(&req); err != nil || req.Email == "" {
			return c.JSON(http.StatusUnprocessableEntity, map[string]any{
				"error": "invalid payload",
			})
		}

		now := time.Now()
		claims := jwt.MapClaims{
			"sub":   req.Email,
			"iss":   s.issuer,
			"aud":   "vote-app",
			"iat":   now.Unix(),
			"exp":   now.Add(30 * time.Minute).Unix(),
			"scope": "read write",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = s.keyID
		signed, err := token.SignedString(s.priv)
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

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
