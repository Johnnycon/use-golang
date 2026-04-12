# Project: Go + GraphQL + Docker Chat App

Real-time chat app: Go API (gqlgen + WebSocket subscriptions), Go web server, PostgreSQL, River job queue, Traefik reverse proxy.

## URLs

All HTTP services are behind Traefik (port 8888):

- Chat UI: `localhost:8888`
- API: `localhost:8888/api/` (Traefik strips `/api` prefix)
- River UI: `riverui.localhost:8888` (host-based — SPAs can't use path prefix stripping)
- Traefik Dashboard: `localhost:8089`
- Postgres: `localhost:5433` (direct, not proxied — TCP protocol)
- Ports 3000–3010 are reserved — do not use.

## Running

```bash
make dev       # development with hot reload (use this for iterating)
make build     # production build
make down      # stop + cleanup
make clean     # stop + delete all data (volumes)
```

**Always use `make dev` during development.** Air watches for file changes and recompiles automatically. Only use `make build` for testing the production image.

## gqlgen Docker Build (Chicken-and-Egg Problem)

gqlgen generates `generated.go` from the schema. Files like `server.go` and `schema.resolvers.go` import types from it. But `generated.go` doesn't exist during `go mod tidy`.

**Solution: 3-stage Dockerfile (`api/Dockerfile`).**

- **Stage 1 (gqlgen):** Only copies files that do NOT reference generated types — `go.mod`, `tools.go`, `gqlgen.yml`, `schema.graphqls`, `resolver.go`, `model/model.go`, `db/`. Strips River deps from go.mod before `go mod tidy` (they're not needed here).
- **Stage 2 (builder):** Copies ALL source files + generated files from stage 1.
- **Stage 3 (scratch):** Just the binary.

**Critical rule:** Never add `server.go`, `schema.resolvers.go`, or `jobs/` to stage 1. If a file imports generated types or packages not available in stage 1, it breaks the build.

## Resolver ↔ Jobs Decoupling

`resolver.go` is copied into the gqlgen stage. It must NOT import `jobs` or `river` packages. Job insertion uses a function field (`InsertJob func(messageID string) error`) wired in `server.go`. This keeps the import graph clean for code generation.

## Model Binding

`Message`, `Room`, `JobResult`, and `CalorieQuery` are defined in `graph/model/model.go` and bound in `gqlgen.yml`. This avoids duplicate type generation and lets `resolver.go` reference these types during the gqlgen stage.

## Job Queue Pattern

Jobs store only reference IDs (e.g. `message_id`), not data. The worker fetches the full record from the DB when it runs. See `api/jobs/process_message.go`.

## Traefik

Scoped to this project via `traefik.project=gotesting` label constraint. This prevents Traefik from picking up containers from other compose projects on the same Docker daemon.

## LLM Integration

`api/llm/llm.go` is a unified LLM client supporting OpenAI (gpt-5.4, gpt-5.4-mini, gpt-5.4-nano) and Google Gemini models. OpenAI calls support a `reasoning_effort` parameter (none/low/medium/high/xhigh) passed via the `shared.ReasoningEffort` type from the `openai-go/v3` SDK. Gemini calls ignore reasoning effort. The client is wired into the calorie estimation worker via a function field (`CallLLM`) to maintain the resolver ↔ jobs decoupling.

## Key Files

- `api/graph/schema.graphqls` — GraphQL schema (source of truth for types/operations)
- `api/graph/resolver.go` — Resolver struct, subscriber maps, HandleJobComplete
- `api/graph/schema.resolvers.go` — Resolver implementations (auto-regenerated header by gqlgen)
- `api/server.go` — Main entry: DB, River, GraphQL wiring
- `api/jobs/process_message.go` — River worker (chat message processing)
- `api/jobs/estimate_calories.go` — River worker (LLM calorie estimation)
- `api/llm/llm.go` — LLM client (OpenAI + Gemini)
- `api/db/db.go` — Database layer + migrations
- `web/templates/calories.html` — Calorie counter UI (model selection, reasoning effort, results with filter/delete)
