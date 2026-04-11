# Chat Room

A real-time chat app built with **Go**, **GraphQL**, **WebSocket subscriptions**, **PostgreSQL**, **River job queue**, and **Docker**, fronted by **Traefik** as a reverse proxy.

```
                         ┌─────────────────────┐
                         │   Traefik (:8888)    │
                         │   reverse proxy      │
                         └──┬───────┬───────┬───┘
                            │       │       │
           localhost:8888/  │       │       │ riverui.localhost:8888
                            ▼       │       ▼
                    ┌───────────┐   │   ┌───────────┐
                    │  Web (Go) │   │   │ River UI  │
                    │  :3080    │   │   │ :8080     │
                    └───────────┘   │   └───────────┘
                                    │ localhost:8888/api/
                                    ▼
                    ┌──────────────────────────────┐
                    │   API (Go + gqlgen)           │
                    │   GraphQL + WebSocket         │
                    │   :8080                       │
                    └──────────┬───────────────────┘
                               │
                    ┌──────────▼───────────────────┐
                    │   PostgreSQL                  │
                    │   messages, rooms, river_job  │
                    │   :5432 (5433 on host)        │
                    └──────────────────────────────┘
```

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed and running
- That's it! Go is not needed on your machine.

## Quick Start

```bash
make dev
open http://localhost:8888
```

Enter a name, create or join a room, and start chatting. Messages appear in real time, persist across restarts, and each message triggers an async job that notifies the sender via toast when complete.

**Stop:** `Ctrl+C` or `make down`

## URLs

| Service | URL |
|---|---|
| Chat UI | http://localhost:8888 |
| GraphQL Playground | http://localhost:8888/api/ |
| River UI (job dashboard) | http://riverui.localhost:8888 |
| Traefik Dashboard | http://localhost:8089 |
| Postgres (direct) | `localhost:5433` (user: `chat`, pass: `chat`) |

## Project Structure

```
├── docker-compose.yml              # All services + Traefik routing labels
├── docker-compose.dev.yml          # Dev overrides (Air hot reload, volume mounts)
├── Makefile                        # make dev / build / down / clean / logs
├── api/
│   ├── Dockerfile                  # 3-stage: gqlgen → builder → scratch
│   ├── Dockerfile.dev              # Dev image with Air
│   ├── server.go                   # Entry point — wires DB, River, GraphQL
│   ├── graph/
│   │   ├── schema.graphqls         # GraphQL schema (source of truth)
│   │   ├── resolver.go             # Resolver struct, WebSocket fan-out
│   │   ├── schema.resolvers.go     # Query/mutation/subscription implementations
│   │   └── model/model.go          # Hand-written types (bound in gqlgen.yml)
│   ├── jobs/
│   │   └── process_message.go      # River worker — fetches message by ID, processes it
│   └── db/
│       ├── db.go                   # Connection pool, queries, migrations
│       └── migrations/001_init.sql # Schema: rooms + messages tables
└── web/
    ├── main.go                     # Serves HTML, injects API/WS URLs
    └── templates/index.html        # Chat UI — vanilla JS, WebSocket subscriptions
```

## How It Works

### Real-Time Chat

1. Browser opens a WebSocket to the API through Traefik
2. Subscribes to `messageSent(room)` — receives new messages pushed from the server
3. Mutations (`sendMessage`) save to Postgres, fan out to all subscribers in the room

### Async Job Processing (River)

1. `sendMessage` enqueues a River job with just the `message_id`
2. The worker fetches the full message from Postgres, simulates processing (2-5s delay)
3. On completion, the result is pushed via `jobCompleted` subscription — **only to the sender**, not broadcast
4. The browser displays a toast notification with the result

### Traefik Reverse Proxy

All HTTP traffic enters through Traefik on port 8888:
- **Path-based routing:** `/` → web, `/api/` → API (prefix stripped)
- **Host-based routing:** `riverui.localhost` → River UI (SPAs with internal `/api/` paths break with prefix stripping)
- Scoped to this project via label constraint (`traefik.project=gotesting`)
- WebSocket connections are proxied transparently (Traefik handles the HTTP → WS upgrade)

### Code Generation (gqlgen)

Schema-first: write `schema.graphqls`, run `gqlgen generate`, implement resolvers. The production Dockerfile handles generation in a separate build stage to solve the chicken-and-egg problem (generated code must exist before compilation, but compilation must succeed before generation can run in the same context).

## Commands

```bash
make dev      # Start with hot reload (Air auto-recompiles on save)
make build    # Start with production build (scratch images, ~10 MB each)
make down     # Stop all containers
make logs     # Follow container logs
make clean    # Stop + wipe database (deletes Postgres volume)
```

| | `make dev` | `make build` |
|---|---|---|
| Base image | Full Go (~250 MB, shared) | `scratch` (~10 MB per service) |
| On code change | Air auto-recompiles (~1-2s) | Must re-run `make build` |
| Best for | Active development | Testing production builds |

## Debug

```bash
docker compose logs -f            # all logs
docker compose logs api           # API only
docker compose ps                 # container status

# Query the API through Traefik
curl -s http://localhost:8888/api/query \
  -H "Content-Type: application/json" \
  -d '{"query": "{ rooms { id name } }"}' | python3 -m json.tool

# Check job queue
docker compose exec postgres psql -U chat -d chat \
  -c "SELECT id, kind, state, args FROM river_job ORDER BY created_at DESC LIMIT 5;"
```

## Common Issues

| Problem | Solution |
|---------|----------|
| `port is already allocated` | `lsof -i :<port>` to find conflicts |
| Traefik can't reach Docker socket | Check Docker Desktop version; ensure v3.6+ Traefik image |
| WebSocket not connecting | Check browser console; Traefik handles WS upgrade automatically |
| Data gone after restart | Use `make down` (keeps data), not `make clean` (deletes volume) |
| API slow on first `make dev` | First run downloads Go deps inside the container; subsequent runs use cache |
