-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS votes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    candidate_id BIGINT NOT NULL,
    voted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT votes_unique_user_candidate UNIQUE (user_id, candidate_id)
);

CREATE TABLE IF NOT EXISTS totals_sharded (
    candidate_id BIGINT NOT NULL,
    bucket INT NOT NULL,
    cnt BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (candidate_id, bucket)
);

CREATE OR REPLACE VIEW totals AS
SELECT
    candidate_id,
    SUM(cnt) AS count
FROM totals_sharded
GROUP BY candidate_id
ORDER BY candidate_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS totals;
DROP TABLE IF EXISTS totals_sharded;
DROP TABLE IF EXISTS votes;
-- +goose StatementEnd
