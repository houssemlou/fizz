# confluent-kafka-go builds librdkafka from bundled source — no system librdkafka needed.
# -tags musl tells the bundled build it is targeting musl (Alpine) instead of glibc.
FROM golang:1.25-alpine AS builder
RUN apk --no-cache add \
    gcc g++ musl-dev make bash pkgconf \
    openssl-dev zstd-dev lz4-dev cyrus-sasl-dev zlib-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -ldflags="-w -s" -o /usr/local/bin/server ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -tags musl -ldflags="-w -s" -o /usr/local/bin/stats-worker ./cmd/stats-worker

FROM alpine:3.21 AS server
RUN apk --no-cache add ca-certificates tzdata libssl3 zstd-libs lz4-libs cyrus-sasl
COPY --from=builder /usr/local/bin/server /usr/local/bin/server
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/server"]

FROM alpine:3.21 AS worker
RUN apk --no-cache add ca-certificates tzdata libssl3 zstd-libs lz4-libs cyrus-sasl
COPY --from=builder /usr/local/bin/stats-worker /usr/local/bin/stats-worker
ENTRYPOINT ["/usr/local/bin/stats-worker"]
