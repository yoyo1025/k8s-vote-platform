package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Config describes the external dependencies and runtime configuration of the worker.
type Config struct {
	RedisAddr      string
	RedisUsername  string
	RedisPassword  string
	RedisStream    string
	RedisGroup     string
	RedisConsumer  string
	ResultsChannel string

	BatchSize     int
	BlockInterval time.Duration
	IdleTimeout   time.Duration

	PGConnString   string
	TotalsBucketID int
}

// GenerateConsumerID returns a best-effort unique consumer identifier.
func GenerateConsumerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker"
	}
	return fmt.Sprintf("%s-%d", host, time.Now().UnixNano())
}

// Processor consumes votes from Redis Streams and persists them to PostgreSQL.
type Processor struct {
	cfg       Config
	log       *log.Logger
	redis     *redis.Client
	pg        *pgxpool.Pool
	lastClaim time.Time
}

// NewProcessor validates connectivity and ensures the consumer group exists.
func NewProcessor(ctx context.Context, cfg Config) (*Processor, error) {
	if cfg.RedisAddr == "" {
		return nil, errors.New("redis addr is required")
	}
	if cfg.RedisStream == "" {
		return nil, errors.New("redis stream is required")
	}
	if cfg.RedisGroup == "" {
		return nil, errors.New("redis group is required")
	}
	if cfg.RedisConsumer == "" {
		cfg.RedisConsumer = GenerateConsumerID()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.BlockInterval <= 0 {
		cfg.BlockInterval = 5 * time.Second
	}
	if cfg.IdleTimeout < 0 {
		cfg.IdleTimeout = 0
	}
	if cfg.PGConnString == "" {
		return nil, errors.New("postgres connection string is required")
	}
	if cfg.ResultsChannel == "" {
		cfg.ResultsChannel = "results:totals"
	}

	logger := log.New(os.Stdout, "[worker] ", log.LstdFlags|log.Lmicroseconds|log.Lmsgprefix)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	// Ensure the consumer group exists.
	if err := ensureConsumerGroup(ctx, rdb, cfg.RedisStream, cfg.RedisGroup); err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, cfg.PGConnString)
	if err != nil {
		return nil, fmt.Errorf("pg connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg ping: %w", err)
	}

	return &Processor{
		cfg:       cfg,
		log:       logger,
		redis:     rdb,
		pg:        pool,
		lastClaim: time.Now(),
	}, nil
}

// Run blocks until the context is cancelled or a fatal error occurs.
func (p *Processor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		entries, err := p.readBatch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			p.log.Printf("read batch error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if len(entries) == 0 {
			if p.cfg.IdleTimeout > 0 && time.Since(p.lastClaim) >= p.cfg.IdleTimeout {
				claimed, err := p.claimIdle(ctx)
				if err != nil {
					p.log.Printf("claim idle error: %v", err)
					time.Sleep(time.Second)
					continue
				}
				if len(claimed) == 0 {
					continue
				}
				if err := p.processBatch(ctx, claimed); err != nil {
					p.log.Printf("process claimed batch error: %v", err)
					time.Sleep(time.Second)
				}
			}
			continue
		}

		if err := p.processBatch(ctx, entries); err != nil {
			p.log.Printf("process batch error: %v", err)
			time.Sleep(time.Second)
			continue
		}
	}
}

// Close releases external resources held by the processor.
func (p *Processor) Close() {
	if p.pg != nil {
		p.pg.Close()
	}
	if p.redis != nil {
		if err := p.redis.Close(); err != nil {
			p.log.Printf("redis close error: %v", err)
		}
	}
}

type voteEntry struct {
	id          string
	userID      int64
	candidateID int64
	votedAt     time.Time
}

func (p *Processor) readBatch(ctx context.Context) ([]voteEntry, error) {
	args := &redis.XReadGroupArgs{
		Group:    p.cfg.RedisGroup,
		Consumer: p.cfg.RedisConsumer,
		Streams:  []string{p.cfg.RedisStream, ">"},
		Count:    int64(p.cfg.BatchSize),
		Block:    p.cfg.BlockInterval,
	}

	streams, err := p.redis.XReadGroup(ctx, args).Result()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		if strings.Contains(err.Error(), "NOGROUP") {
			if err := ensureConsumerGroup(ctx, p.redis, p.cfg.RedisStream, p.cfg.RedisGroup); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return nil, err
	}

	var entries []voteEntry
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			entry, err := parseMessage(msg)
			if err != nil {
				p.log.Printf("skip malformed message %s: %v", msg.ID, err)
				// acknowledge malformed message to avoid infinite loop
				if ackErr := p.redis.XAck(ctx, p.cfg.RedisStream, p.cfg.RedisGroup, msg.ID).Err(); ackErr != nil {
					p.log.Printf("failed to ack malformed message %s: %v", msg.ID, ackErr)
				}
				continue
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func parseMessage(msg redis.XMessage) (voteEntry, error) {
	var entry voteEntry
	entry.id = msg.ID

	userStr, ok := msg.Values["user_id"]
	if !ok {
		return entry, errors.New("missing user_id")
	}
	candStr, ok := msg.Values["candidate_id"]
	if !ok {
		return entry, errors.New("missing candidate_id")
	}

	userID, err := parseInt64(userStr)
	if err != nil {
		return entry, fmt.Errorf("invalid user_id: %w", err)
	}
	candidateID, err := parseInt64(candStr)
	if err != nil {
		return entry, fmt.Errorf("invalid candidate_id: %w", err)
	}

	entry.userID = userID
	entry.candidateID = candidateID
	entry.votedAt = time.Now().UTC()

	if tsVal, ok := msg.Values["ts"]; ok {
		if tsStr, ok := tsVal.(string); ok {
			if ts, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
				entry.votedAt = ts
			}
		}
	}

	return entry, nil
}

func parseInt64(v any) (int64, error) {
	switch t := v.(type) {
	case string:
		return strconv.ParseInt(t, 10, 64)
	case []byte:
		return strconv.ParseInt(string(t), 10, 64)
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

func (p *Processor) processBatch(ctx context.Context, entries []voteEntry) error {
	tx, err := p.pg.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	increments := make(map[int64]int64)
	ackIDs := make([]string, 0, len(entries))

	for _, entry := range entries {
		tag, err := tx.Exec(ctx, `
			INSERT INTO votes (user_id, candidate_id, voted_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, candidate_id) DO NOTHING
		`, entry.userID, entry.candidateID, entry.votedAt)
		if err != nil {
			return fmt.Errorf("insert vote %s: %w", entry.id, err)
		}
		if tag.RowsAffected() == 0 {
			ackIDs = append(ackIDs, entry.id)
			continue
		}

		increments[entry.candidateID]++
		ackIDs = append(ackIDs, entry.id)
	}

	for candidateID, inc := range increments {
		if _, err := tx.Exec(ctx, `
			INSERT INTO totals_sharded (candidate_id, bucket, cnt)
			VALUES ($1, $2, $3)
			ON CONFLICT (candidate_id, bucket)
			DO UPDATE SET cnt = totals_sharded.cnt + EXCLUDED.cnt
		`, candidateID, p.cfg.TotalsBucketID, inc); err != nil {
			return fmt.Errorf("update totals candidate %d: %w", candidateID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit batch: %w", err)
	}

	if len(ackIDs) > 0 {
		if err := p.redis.XAck(ctx, p.cfg.RedisStream, p.cfg.RedisGroup, ackIDs...).Err(); err != nil {
			return fmt.Errorf("ack messages: %w", err)
		}
	}

	if len(increments) > 0 && p.cfg.ResultsChannel != "" {
		if err := p.redis.Publish(ctx, p.cfg.ResultsChannel, "refresh").Err(); err != nil {
			p.log.Printf("publish totals update error: %v", err)
		}
	}

	return nil
}

func ensureConsumerGroup(ctx context.Context, rdb *redis.Client, stream, group string) error {
	if err := rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err(); err != nil {
		if strings.Contains(err.Error(), "BUSYGROUP") {
			return nil
		}
		return fmt.Errorf("create group %s: %w", group, err)
	}
	return nil
}

func (p *Processor) claimIdle(ctx context.Context) ([]voteEntry, error) {
	start := "0-0"
	var claimed []voteEntry

	for {
		msgs, next, err := p.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{
			Stream:   p.cfg.RedisStream,
			Group:    p.cfg.RedisGroup,
			Consumer: p.cfg.RedisConsumer,
			MinIdle:  p.cfg.IdleTimeout,
			Start:    start,
			Count:    int64(p.cfg.BatchSize),
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				break
			}
			return nil, err
		}

		start = next

		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			entry, err := parseMessage(msg)
			if err != nil {
				p.log.Printf("skip malformed claimed message %s: %v", msg.ID, err)
				if ackErr := p.redis.XAck(ctx, p.cfg.RedisStream, p.cfg.RedisGroup, msg.ID).Err(); ackErr != nil {
					p.log.Printf("failed to ack malformed claimed message %s: %v", msg.ID, ackErr)
				}
				continue
			}
			claimed = append(claimed, entry)
		}

		if next == "0-0" {
			break
		}
	}

	p.lastClaim = time.Now()
	return claimed, nil
}
