-- +goose Up
CREATE TABLE orders (
    id          TEXT        PRIMARY KEY,
    customer_id TEXT        NOT NULL,
    items       JSONB       NOT NULL,
    total_cents BIGINT      NOT NULL CHECK (total_cents >= 0),
    status      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    version     INTEGER     NOT NULL
);

CREATE TABLE outbox (
    seq          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    id           TEXT        NOT NULL UNIQUE,
    type         TEXT        NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    aggregate_id TEXT        NOT NULL,
    version      INTEGER     NOT NULL,
    payload      JSONB       NOT NULL,
    published_at TIMESTAMPTZ
);

-- The relay only ever looks at unpublished rows: a partial index stays tiny.
CREATE INDEX outbox_unpublished ON outbox (seq) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE outbox;
DROP TABLE orders;
