# AGENTS.md

Guidance for AI coding agents and contributors working in this repository.

## What this project is

`mcp-api` (Go module `github.com/arthurwerle/mcp-api`) is an
[MCP](https://modelcontextprotocol.io/) server built on
[`github.com/mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go). It exposes
tools over a personal-finance **transactions** dataset by proxying requests to a
backend "transaction service" REST API. Most tools read data; a set of
`create_*` / `update_*` tools also write to the backend. The server itself holds
no data — it is a translation layer between MCP clients and that backend.

## Architecture

The whole server is a few files in `package main`:

- **`main.go`** — process entry point and wiring.
  - Reads configuration from env vars (`getEnv`).
  - Builds a `TransactionClient` and an `MCPServer`, then calls `RegisterTools`.
  - Selects a transport:
    - `stdio` → `server.ServeStdio(s)` (used by Claude Desktop).
    - `http` → `server.NewSSEServer(...)` on `:SERVER_PORT`, plus a plain
      `/health` HTTP handler (`healthHandler`).
  - Defines `defaultInstructions`: a long string passed via
    `server.WithInstructions(...)`. This is the closest thing the MCP server has
    to a **system prompt** — ambient context describing the data model
    (transactions, income/expense, recurring vs. non-recurring) and tool
    guidance. Override at runtime with `MCP_INSTRUCTIONS`.
- **`tools.go`** — tool definitions.
  - `RegisterTools` calls one `registerXxx` function per tool.
  - Each `registerXxx` builds an `mcp.NewTool(...)` (name, description,
    parameters) and registers a handler that parses params, calls the matching
    `TransactionClient` method, and wraps the result via `toolResult`.
  - `toolResult` is the shared helper: turns `([]byte, error)` into an MCP
    success/error result and logs the outcome.
- **`client.go`** — `TransactionClient`, a thin HTTP client for the backend.
  - `get` / `getURL` perform GET requests, log timing, and surface non-2xx
    responses as errors. `send` (and its `post` / `put` wrappers) does the same
    for requests with a JSON body. Read tools map to GETs; `create_*` / `update_*`
    tools map to POST / PUT against backend paths (`/transactions`, `/categories`,
    `/subcategories`, `/locations`, etc.).
  - `HealthCheck` hits `/health` at the backend root (strips the `/api/v2`
    suffix), not under the API base path.

## File map

| Path | Purpose |
| --- | --- |
| `main.go` | Entry point, config, transport selection, default instructions, health endpoint |
| `tools.go` | MCP tool registration and handlers |
| `client.go` | HTTP client for the backend transaction service |
| `Makefile` | Build / run / test / lint / Docker targets |
| `Dockerfile` | Multi-stage build → `alpine` image |
| `docker-compose.yml` | Local Docker run (HTTP mode, port `6666:3006`) |
| `docker-compose.staging.yml` | Staging compose (variables from `stack.env`) |
| `stack.env` | Default env for Docker/compose |
| `.github/workflows/build.yml` | CI: cross-compile binaries as artifacts |

## Common commands

```bash
make build   # go build -o bin/mcp-api .
make run     # go run .
make test    # go test -v ./... -count=1
make fmt     # gofmt -s -w . && go mod tidy
make lint    # golangci-lint run ./... (no-op if not installed)
make deps    # go mod download && go mod tidy
make clean   # rm -rf bin/

# Docker
make docker-up / docker-down / docker-logs
make compose-up            # docker compose up --build
make staging-up / staging-down / staging-logs
```

## Configuration (env vars)

| Variable | Default | Notes |
| --- | --- | --- |
| `TRANSPORT` | `http` | `stdio` or `http` |
| `SERVER_PORT` | `3006` | HTTP/SSE port |
| `TRANSACTION_SERVICE_URL` | `http://localhost:1235/api/v2` | Backend base URL |
| `LOG_LEVEL` | `info` | `info` or `debug` |
| `MCP_INSTRUCTIONS` | `defaultInstructions` | Overrides server instructions |

In HTTP mode the SSE endpoint is `/sse` and a JSON `/health` endpoint reports the
service status and backend reachability.

## Conventions

- **Go formatting:** run `make fmt` (gofmt `-s`) before committing; keep
  `go.mod`/`go.sum` tidy.
- **Logging:** structured logging via `log/slog` as JSON to stderr. Use the
  passed-in `*slog.Logger`; log tool calls and upstream requests/responses.
- **Read vs. write tools:** read tools map to a backend GET. Write tools
  (`create_*` / `update_*`) map to POST / PUT and must be tagged with
  `mcp.WithReadOnlyHintAnnotation(false)` so clients know they have side effects.
  There are intentionally no delete tools. Build write payloads as a
  `map[string]any` containing only the fields the caller provided (see `pickArgs`
  in `tools.go`) so partial updates leave other fields untouched.
- **Error handling:** return tool errors via `mcp.NewToolResultError(...)`
  (handled by `toolResult`) rather than returning a Go `error` from the handler,
  so the model sees a usable message.

## Adding a new tool

1. Add a method to `TransactionClient` in `client.go` that performs the backend
   request (reuse `get` / `getURL` for reads, `post` / `put` for writes).
2. Add a `registerYourTool(s, client, logger)` function in `tools.go`: define the
   `mcp.NewTool` with name/description/params, parse params in the handler, call
   your client method, and return `toolResult(body, err, logger, "your_tool")`.
3. Wire it into `RegisterTools` in `tools.go`.

## CI & deployment

- **CI** (`.github/workflows/build.yml`): on push to `main` (or manual dispatch),
  upgrades `mcp-go` to latest, then cross-compiles
  `mcp-transactions-darwin-arm64` and `mcp-transactions-linux-amd64`, uploaded as
  workflow artifacts (30-day retention). These binaries are what users install
  into Claude Desktop.
- **Deployment:** runs in HTTP mode via Docker Compose. `docker-compose.yml`
  joins an external `financer-transactions_transactions-network` so the server
  can reach the backend service by container name.

## Gotchas

- Claude Desktop integration only works with the **binary + `stdio` transport**;
  the HTTP/SSE transport is intended for networked/custom-agent use.
- `HealthCheck` deliberately strips `/api/v2` from `TRANSACTION_SERVICE_URL`
  because the backend's `/health` lives at the root.
