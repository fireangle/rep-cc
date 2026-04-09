# Free AI API Proxy (Go)

## Overview

A free AI reverse proxy written in Go that exposes OpenAI and Anthropic API formats, backed by Replit AI Integrations. No personal API keys needed — usage is billed to your Replit credits. Protect your proxy with a `PROXY_API_KEY` secret.

## Stack

- **Language**: Go 1.22+ (pure Go, no Node.js)
- **Runtime**: Single binary, no external dependencies

## Project Structure

```
artifacts/go-proxy/
  main.go       — Entry point, routing, auth middleware
  proxy.go      — Request forwarding logic
  translate.go  — Format translation (OpenAI ↔ Anthropic)
  types.go      — Shared type definitions
  go.mod        — Go module definition
```

## Proxy Endpoints

All endpoints require `Authorization: Bearer <PROXY_API_KEY>` or `x-api-key` header, except `/health`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (no auth) |
| GET | `/v1/models` | List all available models |
| POST | `/v1/chat/completions` | OpenAI-compatible chat (streaming + tools) |
| POST | `/v1/messages` | Anthropic Messages API (streaming + tools) |

## Supported Models

**OpenAI:** `gpt-5.2`, `gpt-5-mini`, `gpt-5-nano`, `o4-mini`, `o3`  
**Anthropic:** `claude-opus-4-6`, `claude-sonnet-4-6`, `claude-haiku-4-5`

## Secrets / Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `PROXY_API_KEY` | Yes | Secret key clients must send to authenticate |
| `AI_INTEGRATIONS_OPENAI_BASE_URL` | Auto | Set by Replit OpenAI integration |
| `AI_INTEGRATIONS_OPENAI_API_KEY` | Auto | Set by Replit OpenAI integration |
| `AI_INTEGRATIONS_ANTHROPIC_BASE_URL` | Auto | Set by Replit Anthropic integration |
| `AI_INTEGRATIONS_ANTHROPIC_API_KEY` | Auto | Set by Replit Anthropic integration |
| `PORT` | Auto | Port to listen on (default: 8080) |

## Deployment

- **Build**: `go build -o proxy .`
- **Run**: `./proxy`
- **Health check**: `GET /health`

## Usage Example

```bash
# Health check (no auth)
curl https://your-domain.replit.app/health

# List models
curl https://your-domain.replit.app/v1/models \
  -H "Authorization: Bearer YOUR_PROXY_API_KEY"

# Chat (OpenAI format)
curl https://your-domain.replit.app/v1/chat/completions \
  -H "Authorization: Bearer YOUR_PROXY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.2","messages":[{"role":"user","content":"Hello"}]}'

# Chat (Anthropic format)
curl https://your-domain.replit.app/v1/messages \
  -H "x-api-key: YOUR_PROXY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-6","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}]}'
```
