package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
)

const (
	defaultRedisChannel  = "results:totals"
	defaultHeartbeatFreq = 30 * time.Second
)

// Config contains the external dependencies required by the result-query service.
type Config struct {
	PGConnString string

	RedisAddr     string
	RedisUsername string
	RedisPassword string
	RedisChannel  string
}

// Server implements the gRPC ResultService backed by Postgres totals and Redis notifications.
type Server struct {
	resultv1.UnimplementedResultServiceServer

	pool    *pgxpool.Pool
	redis   *redis.Client
	channel string
	logger  *log.Logger
}

// New initialises connections to Postgres and Redis and returns a ready Server.
func New(ctx context.Context, cfg Config) (*Server, error) {
	if cfg.PGConnString == "" {
		return nil, errors.New("pg connection string is required")
	}
	if cfg.RedisAddr == "" {
		return nil, errors.New("redis addr is required")
	}
	if cfg.RedisChannel == "" {
		cfg.RedisChannel = defaultRedisChannel
	}

	logger := log.New(log.Writer(), "[result-query] ", log.LstdFlags|log.Lmsgprefix)

	pool, err := pgxpool.New(ctx, cfg.PGConnString)
	if err != nil {
		return nil, fmt.Errorf("pg connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg ping: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Server{
		pool:    pool,
		redis:   redisClient,
		channel: cfg.RedisChannel,
		logger:  logger,
	}, nil
}

// Close releases underlying resources.
func (s *Server) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			s.logger.Printf("redis close error: %v", err)
		}
	}
}

// GetTotals returns the latest aggregated totals from Postgres.
func (s *Server) GetTotals(ctx context.Context, _ *resultv1.GetTotalsRequest) (*resultv1.GetTotalsResponse, error) {
	totals, updatedAt, err := s.fetchTotals(ctx)
	if err != nil {
		return nil, err
	}
	return &resultv1.GetTotalsResponse{
		Totals:    totals,
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}, nil
}

// SubscribeTotals streams totals whenever the worker publishes an update event.
func (s *Server) SubscribeTotals(req *resultv1.SubscribeTotalsRequest, stream resultv1.ResultService_SubscribeTotalsServer) error {
	ctx := stream.Context()

	// Send initial snapshot.
	totals, updatedAt, err := s.fetchTotals(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&resultv1.SubscribeTotalsResponse{
		Totals:    totals,
		UpdatedAt: updatedAt.Format(time.RFC3339),
	}); err != nil {
		return err
	}

	pubsub := s.redis.Subscribe(ctx, s.channel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	ch := pubsub.Channel()
	heartbeat := time.NewTicker(defaultHeartbeatFreq)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeat.C:
			// Periodic heartbeat with a refresh to keep the stream warm.
			totals, updatedAt, err := s.fetchTotals(ctx)
			if err != nil {
				s.logger.Printf("heartbeat fetch error: %v", err)
				continue
			}
			if err := stream.Send(&resultv1.SubscribeTotalsResponse{
				Totals:    totals,
				UpdatedAt: updatedAt.Format(time.RFC3339),
			}); err != nil {
				return err
			}
		case msg, ok := <-ch:
			if !ok {
				return errors.New("redis pubsub channel closed")
			}
			if msg == nil {
				continue
			}

			totals, updatedAt, err := s.fetchTotals(ctx)
			if err != nil {
				s.logger.Printf("fetch totals error: %v", err)
				continue
			}
			if err := stream.Send(&resultv1.SubscribeTotalsResponse{
				Totals:    totals,
				UpdatedAt: updatedAt.Format(time.RFC3339),
			}); err != nil {
				return err
			}
		}
	}
}

func (s *Server) fetchTotals(ctx context.Context) ([]*resultv1.Totals, time.Time, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT candidate_id, count
		FROM totals
		ORDER BY candidate_id`)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("query totals: %w", err)
	}
	defer rows.Close()

	var totals []*resultv1.Totals
	for rows.Next() {
		var (
			candidateID int64
			count       int64
		)
		if err := rows.Scan(&candidateID, &count); err != nil {
			return nil, time.Time{}, fmt.Errorf("scan totals: %w", err)
		}
		totals = append(totals, &resultv1.Totals{
			CandidateId: uint64(candidateID),
			Count:       uint64(count),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, time.Time{}, fmt.Errorf("rows totals: %w", err)
	}

	var updatedAt time.Time
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(voted_at), NOW())
		FROM votes`).Scan(&updatedAt); err != nil {
		return nil, time.Time{}, fmt.Errorf("query updated_at: %w", err)
	}

	return totals, updatedAt.UTC(), nil
}
