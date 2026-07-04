# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26.1

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/go-tiny-claw ./cmd/claw

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        openssh-client \
        ripgrep \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

COPY --from=builder /out/go-tiny-claw /usr/local/bin/go-tiny-claw

ENV TINY_CLAW_MODE=feishu-ws \
    TINY_CLAW_PROVIDER=minimax \
    MINIMAX_MODEL=MiniMax-M3 \
    MINIMAX_BASE_URL=https://api.minimaxi.com/anthropic \
    TINY_CLAW_HTTP_ADDR=:48080 \
    TINY_CLAW_ENABLE_THINKING=true

ENTRYPOINT ["/usr/local/bin/go-tiny-claw"]
