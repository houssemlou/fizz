CREATE TABLE fizz_requests (
    idempotent_id   UUID        NOT NULL PRIMARY KEY,
    last_request_id UUID        NOT NULL,
    int1            INTEGER     NOT NULL,
    int2            INTEGER     NOT NULL,
    lim             INTEGER     NOT NULL,
    str1            TEXT        NOT NULL,
    str2            TEXT        NOT NULL,
    hits            INTEGER     NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
