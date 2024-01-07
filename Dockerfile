# syntax=docker/dockerfile:1

FROM golang:1.21-bookworm AS build
WORKDIR /app
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,target=/app \
    OUT_DIR=/tmp make yukid

FROM debian:bookworm-slim
RUN apt update && apt install -y sqlite3 && rm -rf /var/lib/apt/lists/*
COPY --link --from=build /tmp/yukid /yukid
CMD ["/yukid"]
