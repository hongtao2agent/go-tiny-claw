# Docker Deployment

This deployment runs `go-tiny-claw` in Feishu WebSocket long-connection mode and keeps MiniMax-M3 as the default model.

## Setup

```bash
cd deploy
cp .env.example .env
```

Edit `.env` and fill:

- `MINIMAX_API_KEY`
- `FEISHU_APP_ID`
- `FEISHU_APP_SECRET`
- `FEISHU_VERIFY_TOKEN`
- `FEISHU_ENCRYPT_KEY`

## Run

```bash
docker compose up -d --build
```

No public callback endpoint is required in WebSocket mode.

## Logs

```bash
docker compose logs -f go-tiny-claw
```

## Notes

- The compose file mounts the repository root to `/workspace`, so file tools and `bash` operate on the deployed project directory.
- `bash`, `git`, `curl`, `openssh-client`, and `ripgrep` are installed in the runtime image for common Agent tasks.
- If `golang:1.26.1-bookworm` is not available, set `GO_VERSION` in `.env` to an available Go image tag compatible with this project.
- If you need the old HTTP webhook mode, set `TINY_CLAW_MODE=feishu`, expose port `48080`, and configure Feishu callback URL `/webhook/event`.
