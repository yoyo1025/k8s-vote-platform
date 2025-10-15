package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
	"github.com/yoyo1025/k8s-vote-platform/services/result-query/internal/server"
	"google.golang.org/grpc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := server.Config{
		PGConnString:  buildPostgresDSN(),
		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
		RedisUsername: os.Getenv("REDIS_USERNAME"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisChannel:  getenv("REDIS_CHANNEL", "results:totals"),
	}

	grpcAddr := getenv("GRPC_ADDR", ":50051")

	srv, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}
	defer srv.Close()

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	resultv1.RegisterResultServiceServer(grpcServer, srv)

	runErr := make(chan error, 1)
	go func() {
		log.Printf("result-query gRPC listening on %s", grpcAddr)
		runErr <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		log.Println("result-query shutdown initiated")
	case err := <-runErr:
		if err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		log.Println("force stopping gRPC server")
		grpcServer.Stop()
	case <-stopped:
		log.Println("gRPC server stopped gracefully")
	}
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
	user := getenv("PG_USER", "vote")
	password := os.Getenv("PG_PASSWORD")
	database := getenv("PG_DATABASE", "vote")
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
