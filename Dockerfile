FROM rust:1.88-slim AS imgparse-builder
WORKDIR /build/imgparse
COPY imgparse/Cargo.toml imgparse/Cargo.lock ./
COPY imgparse/src ./src
COPY imgparse/models ./models
RUN cargo build --release

FROM golang:1.26-bookworm AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o wordle-bot . && \
    CGO_ENABLED=0 go build -o nickcheck ./cmd/nickcheck

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app

COPY --from=imgparse-builder /build/imgparse/target/release/imgparse ./imgparse
COPY --from=imgparse-builder /build/imgparse/models ./models
COPY --from=go-builder /build/wordle-bot ./wordle-bot
COPY --from=go-builder /build/nickcheck ./nickcheck

ENV IMGPARSE_BIN=/app/imgparse \
    IMGPARSE_MODELS_DIR=/app/models \
    RESULTS_FILE=/data/wordle_results.json \
    CURSOR_FILE=/data/cursor.txt \
    DAEMON_CONFIG_FILE=/data/daemon_config.toml

VOLUME ["/data"]
CMD ["./wordle-bot"]
