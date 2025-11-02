package auth

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
)

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Server struct {
	e *echo.Echo
}

func New() (*Server, error) {
	e := echo.New()

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:  "header:X-CSRF-Token",
		CookieName:   "csrf_token",
		CookieSecure: false,
	}))

	if err := loadKeys(); err != nil {
		return nil, fmt.Errorf("failed to load keys: %v", err)
	}

	s := &Server{
		e: e,
	}

	s.routes()
	return s, nil
}

func (s *Server) routes() {
	// Liveness
	s.e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	s.e.POST("/auth/login", func(c echo.Context) error {
		var req loginReq
		if err := c.Bind(&req); err != nil || req.Email == "" {
			return c.JSON(http.StatusUnprocessableEntity, map[string]any{
				"error": "invalid payload",
			})
		}
		claims := jwt.RegisteredClaims{
			Subject:   req.Email,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Issuer:    "auth-service",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
		t, err := token.SignedString(privateKey)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "failed to sign token",
			})
		}

		cookie := new(http.Cookie)
		cookie.Name = "access_token"
		cookie.Value = t
		cookie.Path = "/"
		cookie.HttpOnly = true
		cookie.SameSite = http.SameSiteLaxMode
		cookie.MaxAge = 3600
		cookie.Secure = false
		c.SetCookie(cookie)

		return c.JSON(http.StatusOK, map[string]any{
			"message": "login successful",
		})
	})

	s.e.GET("/auth/me", func(c echo.Context) error {
		cookie, err := c.Cookie("access_token")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"error": "missing access token",
			})
		}

		token, err := jwt.ParseWithClaims(cookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
			return publicKey, nil
		})
		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"error": "invalid access token",
			})
		}

		claims, ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"error": "invalid access token claims",
			})
		}

		return c.JSON(http.StatusOK, map[string]any{
			"email": claims.Subject,
		})
	})

	s.e.GET("/auth/csrf", func(c echo.Context) error {
		csrfToken := c.Get(middleware.DefaultCSRFConfig.ContextKey).(string)
		return c.JSON(http.StatusOK, echo.Map{
			"csrf_token": csrfToken,
		})
	})
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

func loadKeys() error {
	pubPath := getenvDefault("PUBLIC_PEM_PATH", "/run/secrets/public.pem")
	privPath := getenvDefault("PRIVATE_PEM_PATH", "/run/secrets/private.pem")
	publicKeyData, err := os.ReadFile(pubPath)
	if err != nil {
		return err
	}
	publicKey, err = jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return err
	}

	privateKeyData, err := os.ReadFile(privPath)
	if err != nil {
		return err
	}
	privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return err
	}
	return nil
}

func getenvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
