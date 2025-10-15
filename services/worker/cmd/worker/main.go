package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yoyo1025/k8s-vote-platform/services/worker/internal/worker"
)

func main() {
	cfg := worker.Config{
		RedisAddr:      getenv("REDIS_ADDR", "localhost:6379"),
		RedisUsername:  os.Getenv("REDIS_USERNAME"),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),
		RedisStream:    getenv("REDIS_STREAM", "stream:votes"),
		RedisGroup:     getenv("REDIS_GROUP", "tally"),
		RedisConsumer:  getenv("REDIS_CONSUMER", worker.GenerateConsumerID()),
		ResultsChannel: getenv("RESULTS_CHANNEL", "results:totals"),
		BatchSize:      atoiDefault(os.Getenv("BATCH_SIZE"), 100),
		BlockInterval:  durationDefault(os.Getenv("BLOCK_INTERVAL"), 5*time.Second),
		IdleTimeout:    durationDefault(os.Getenv("IDLE_TIMEOUT"), 30*time.Second),
		PGConnString:   buildPostgresDSN(),
		TotalsBucketID: atoiDefault(os.Getenv("TOTALS_BUCKET_ID"), 0),
	}

	initialCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processor, err := worker.NewProcessor(initialCtx, cfg)
	if err != nil {
		log.Fatalf("initialise worker: %v", err)
	}

	ctx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	errCh := make(chan error, 1)
	go func() {
		errCh <- processor.Run(ctx)
	}()

	log.Printf("worker started: stream=%s group=%s consumer=%s batch=%d bucket=%d",
		cfg.RedisStream, cfg.RedisGroup, cfg.RedisConsumer, cfg.BatchSize, cfg.TotalsBucketID)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	log.Println("shutdown signal received")
	cancelRun()
	cancel()

	if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("worker stopped with error: %v", err)
	}
	processor.Close()
	log.Println("worker stopped")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoiDefault(v string, def int) int {
	if v == "" {
		return def
	}
	if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
		return parsed
	}
	return def
}

func durationDefault(v string, def time.Duration) time.Duration {
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	if d <= 0 {
		return def
	}
	return d
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
