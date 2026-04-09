# Free AI API Proxy (Go)

## Overview

A free AI reverse proxy written in Go that exposes OpenAI and Anthropic API formats, backed by Replit AI Integrations. No personal API keys needed — usage is billed to your Replit credits. Protect your proxy with a `PROXY_API_KEY` secret.

## Stack

- **Language**: Go 1.22+
- **Runtime**: Single binary, no dependencies

## Proxy Endpoints

All proxy endpoints require `Authorization: Bearer <PROXY_API_KEY>` or `x-api-key` header.

- `GET /health` — health check (no auth required)
- `GET /v1/models` — list all available models
- `POST /v1/chat/completions` — OpenAI-compatible endpoint (streaming + tool calls)
- `POST /v1/messages` — Anthropic Messages API native format (streaming + tool calls)

## Supported Models

**OpenAI:** `gpt-5.2`, `gpt-5-mini`, `gpt-5-nano`, `o4-mini`, `o3`
**Anthropic:** `claude-opus-4-6`, `claude-sonnet-4-6`, `claude-haiku-4-5`

## Secrets / Environment Variables

- `PROXY_API_KEY` — Required. A secret key clients must send to authenticate with the proxy.
- `AI_INTEGRATIONS_OPENAI_BASE_URL` / `AI_INTEGRATIONS_OPENAI_API_KEY` — Auto-configured by Replit AI Integrations.
- `AI_INTEGRATIONS_ANTHROPIC_BASE_URL` / `AI_INTEGRATIONS_ANTHROPIC_API_KEY` — Auto-configured by Replit AI Integrations.
- `PORT` — Port to listen on (default: 8080).

## Files

- `artifacts/go-proxy/main.go` — Entry point, routing, auth middleware
- `artifacts/go-proxy/proxy.go` — Request forwarding logic
- `artifacts/go-proxy/translate.go` — Format translation (OpenAI ↔ Anthropic)
- `artifacts/go-proxy/types.go` — Shared type definitions

## Usage Example

```bash
# Health check
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
