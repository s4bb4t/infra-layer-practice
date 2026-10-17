-- +goose Up
CREATE TABLE orders (
    id          VARCHAR(64)  NOT NULL PRIMARY KEY,
    customer_id VARCHAR(64)  NOT NULL,
    items       JSON         NOT NULL,
    total_cents BIGINT       NOT NULL,
    status      VARCHAR(16)  NOT NULL,
    created_at  DATETIME(6)  NOT NULL,
    version     INT          NOT NULL,
    INDEX orders_customer_created (customer_id, created_at DESC)
) ENGINE = InnoDB;

CREATE TABLE outbox (
    seq          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    id           VARCHAR(64)  NOT NULL UNIQUE,
    type         VARCHAR(64)  NOT NULL,
    occurred_at  DATETIME(6)  NOT NULL,
    aggregate_id VARCHAR(64)  NOT NULL,
    version      INT          NOT NULL,
    payload      JSON         NOT NULL,
    published_at DATETIME(6)  NULL,
    INDEX outbox_unpublished (published_at, seq)
) ENGINE = InnoDB;

-- +goose Down
DROP TABLE outbox;
DROP TABLE orders;
