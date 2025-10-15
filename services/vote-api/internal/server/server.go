package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

// Config bundles external dependencies required to run the vote API.
type Config struct {
	RedisAddr     string
	RedisUsername string
	RedisPassword string
	RedisStream   string
	PGConnString  string
}

// Server exposes REST endpoints to accept votes and read aggregates.
type Server struct {
	e      *echo.Echo
	redis  *redis.Client
	pgpool *pgxpool.Pool
	stream string
}

// New wires dependencies and returns a configured Server.
func New(ctx context.Context, cfg Config) (*Server, error) {
	if cfg.RedisAddr == "" {
		return nil, errors.New("redis addr is required")
	}
	if cfg.RedisStream == "" {
		return nil, errors.New("redis stream is required")
	}
	if cfg.PGConnString == "" {
		return nil, errors.New("postgres connection string is required")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.PGConnString)
	if err != nil {
		return nil, fmt.Errorf("pg connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg ping: %w", err)
	}

	e := echo.New()
	s := &Server{
		e:      e,
		redis:  rdb,
		pgpool: pool,
		stream: cfg.RedisStream,
	}
	s.routes()
	return s, nil
}

// Start begins serving HTTP traffic on the provided address.
func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

// Shutdown gracefully stops the HTTP server and releases resources.
func (s *Server) Shutdown(ctx context.Context) error {
	var errs []error
	if err := s.e.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errs = append(errs, err)
	}
	s.pgpool.Close()
	if err := s.redis.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (s *Server) routes() {
	s.e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	s.e.POST("/votes", s.handleVote)
	s.e.GET("/results", s.handleResults)
}

type voteRequest struct {
	UserID      int64 `json:"user_id"`
	CandidateID int64 `json:"candidate_id"`
}

type voteResponse struct {
	Status string `json:"status"`
}

func (s *Server) handleVote(c echo.Context) error {
	var req voteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid payload"})
	}
	if err := validateVoteRequest(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	entry := &redis.XAddArgs{
		Stream: s.stream,
		Values: map[string]any{
			"user_id":      strconv.FormatInt(req.UserID, 10),
			"candidate_id": strconv.FormatInt(req.CandidateID, 10),
			"ts":           time.Now().UTC().Format(time.RFC3339Nano),
		},
	}

	if _, err := s.redis.XAdd(c.Request().Context(), entry).Result(); err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "failed to enqueue vote"})
	}

	return c.JSON(http.StatusAccepted, voteResponse{Status: "accepted"})
}

type totalsResponse struct {
	Totals    []candidateTotal `json:"totals"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type candidateTotal struct {
	CandidateID int64 `json:"candidate_id"`
	Count       int64 `json:"count"`
}

func (s *Server) handleResults(c echo.Context) error {
	ctx := c.Request().Context()

	rows, err := s.pgpool.Query(ctx, `SELECT candidate_id, count FROM totals ORDER BY candidate_id`)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "failed to query totals"})
	}
	defer rows.Close()

	var totals []candidateTotal
	for rows.Next() {
		var ct candidateTotal
		if err := rows.Scan(&ct.CandidateID, &ct.Count); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to parse totals"})
		}
		totals = append(totals, ct)
	}
	if err := rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "failed to read totals"})
	}

	var updatedAt time.Time
	if err := s.pgpool.QueryRow(ctx, `SELECT COALESCE(MAX(voted_at), NOW()) FROM votes`).Scan(&updatedAt); err != nil {
		return c.JSON(http.StatusBadGateway, map[string]any{"error": "failed to determine updated time"})
	}

	return c.JSON(http.StatusOK, totalsResponse{
		Totals:    totals,
		UpdatedAt: updatedAt.UTC(),
	})
}

func validateVoteRequest(req voteRequest) error {
	if req.UserID <= 0 {
		return errors.New("user_id must be positive")
	}
	if req.CandidateID <= 0 {
		return errors.New("candidate_id must be positive")
	}
	return nil
}
