package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yoyo1025/k8s-vote-platform/services/vote-api/internal/server"
)

func main() {
	ctx := context.Background()

	cfg := server.Config{
		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
		RedisUsername: os.Getenv("REDIS_USERNAME"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisStream:   getenv("REDIS_STREAM", "stream:votes"),
		PGConnString:  buildPostgresDSN(),
	}
	httpAddr := getenv("HTTP_ADDR", ":8080")

	srv, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(httpAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	log.Printf("vote-api listening on %s", httpAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
	log.Println("vote-api stopped")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func buildPostgresDSN() string {
	if dsn := os.Getenv("PG_DSN"); dsn != "" {
		return dsn
	}

	host := getenv("PG_HOST", "localhost")
	port := getenv("PG_PORT", "5432")
	user := getenv("PG_USER", "app")
	password := os.Getenv("PG_PASSWORD")
	database := getenv("PG_DATABASE", "app")
	sslmode := getenv("PG_SSLMODE", "disable")

	if user == "" || database == "" {
		log.Fatal("PG_USER and PG_DATABASE must be set")
	}

	userEsc := url.QueryEscape(user)
	if password != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			userEsc, url.QueryEscape(password), host, port, database, sslmode)
	}

	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s",
		userEsc, host, port, database, sslmode)
}
