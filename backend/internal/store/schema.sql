CREATE TABLE IF NOT EXISTS trips (
    id          text PRIMARY KEY,
    start_at    timestamptz NOT NULL,
    end_at      timestamptz NOT NULL,
    amount      bigint      NOT NULL CHECK (amount > 0),
    payment     text        NOT NULL CHECK (payment IN ('cash', 'card')),
    commission  bigint      NOT NULL CHECK (commission >= 0 AND commission <= amount),
    -- хэш содержимого: одна и та же поездка не попадёт дважды даже под разными id
    fingerprint text        NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);

CREATE INDEX IF NOT EXISTS trips_start_at_idx ON trips (start_at);
