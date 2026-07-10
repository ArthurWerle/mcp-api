# mcp-api

An [MCP](https://modelcontextprotocol.io/) server that gives an LLM access to a
personal-finance **transactions** dataset. It acts as a thin bridge between an
MCP client (Claude Desktop, a custom agent, etc.) and a backend transaction REST
service, exposing a handful of finance-focused tools. Most tools read data; a set
of `create_*` / `update_*` tools can also write to the backend.

## Tools

### Read

| Tool | Description |
| --- | --- |
| `list_transactions` | List transactions with optional filters (date range, category, type, query, current month, pagination) |
| `get_transaction` | Get a single transaction by ID |
| `get_latest_transactions` | Get the most recent transactions |
| `get_biggest_transactions` | Get the biggest transactions for a month/year |
| `get_average_by_type` | Average amount grouped by type (income/expense) |
| `get_average_by_category` | Average amount grouped by category |
| `list_categories` | List all categories |
| `list_subcategories` | List all subcategories |
| `list_locations` | List all locations (places/merchants) |
| `health_check` | Check the backend transactions service |

### Write

| Tool | Description |
| --- | --- |
| `create_transaction` | Create a transaction (`amount` and `type` required; optional category/subcategory by id, free-text `location`, dates, recurrence). Attributed to user id 1 by default |
| `update_transaction` | Update a transaction by ID (only the fields you pass are changed) |
| `create_category` | Create a category (`name` required; optional `description`, `color`) |
| `update_category` | Update a category by ID |
| `create_subcategory` | Create a subcategory (`name` required; optional `description`, `color`) |
| `update_subcategory` | Update a subcategory by ID |
| `create_location` | Create a location by `name` (deduplicated — returns the existing one if it already exists) |
| `update_location` | Update a location's name by ID |

There are no delete tools.

## Prerequisites

- **Go 1.25+** (only needed to build the binary yourself)
- A running **transaction service** backend that the server talks to, reachable
  via `TRANSACTION_SERVICE_URL`.

## Build the binary

```bash
make build         # produces ./bin/mcp-api
# or, equivalently:
go build -o bin/mcp-api .
```

Prebuilt binaries for **macOS (Apple Silicon)** and **Linux (amd64)** are also
produced by CI on every push to `main` — see the `Build Binaries` workflow
artifacts (`mcp-transactions-darwin-arm64`, `mcp-transactions-linux-amd64`).

## Use with Claude Desktop (binary / stdio)

> **Note:** Claude Desktop only works reliably with the **compiled binary using
> the `stdio` transport**. Pointing Claude Desktop at the HTTP/SSE server has
> not worked — use the binary.

1. Build the binary (see above) and note its **absolute** path, e.g.
   `/Users/you/code/mcp-api/bin/mcp-api`.
2. Open Claude Desktop's config file:
   - **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
3. Add the server under `mcpServers`:

   ```json
   {
     "mcpServers": {
       "mcp-api": {
         "command": "/absolute/path/to/mcp-api/bin/mcp-api",
         "env": {
           "TRANSPORT": "stdio",
           "TRANSACTION_SERVICE_URL": "http://localhost:1235/api/v2"
         }
       }
     }
   }
   ```

4. **Restart Claude Desktop.** The transaction tools will appear in the tools
   menu, and you can ask things like *"What were my biggest expenses last month?"*

### Windows (including building from WSL)

There is no prebuilt Windows binary from CI, and CI only cross-compiles for
macOS/Linux (see the `Build Binaries` workflow), so you need to build one
yourself. If you're building from inside **WSL**, cross-compile a native
Windows binary — a plain `go build` inside WSL produces a Linux ELF, which
Claude Desktop (a native Windows app) cannot execute:

```bash
cd ~/path/to/mcp-api
GOOS=windows GOARCH=amd64 go build -o bin/mcp-api.exe .
```

Claude Desktop resolves `command` on the **Windows** side, so avoid pointing
it at a `\\wsl.localhost\...` UNC path — it's an easy source of JSON
escaping bugs (a UNC path needs *four* backslashes when escaped in JSON,
e.g. `\\\\wsl.localhost\\...`) and is less reliable than a local file. Instead,
copy the binary onto native Windows storage:

```bash
mkdir -p /mnt/c/Users/<your-windows-username>/mcp-api
cp bin/mcp-api.exe /mnt/c/Users/<your-windows-username>/mcp-api/
```

Then edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mcp-api": {
      "command": "C:\\Users\\<your-windows-username>\\mcp-api\\mcp-api.exe",
      "env": {
        "TRANSPORT": "stdio",
        "TRANSACTION_SERVICE_URL": "http://localhost:1235/api/v2"
      }
    }
  }
}
```

Restart Claude Desktop (fully quit from the system tray, not just close the
window). If you update the binary later, re-run the build and re-copy it to
the same Windows path.

## Use with custom agents (LangChain / HTTP + SSE)

For programmatic use from a framework like LangChain, run the server in **HTTP
mode**, which exposes an SSE endpoint and a `/health` check.

1. Start the server (defaults: `TRANSPORT=http`, `SERVER_PORT=3006`):

   ```bash
   TRANSPORT=http TRANSACTION_SERVICE_URL="http://localhost:1235/api/v2" make run
   # or run the binary directly:
   TRANSPORT=http ./bin/mcp-api
   ```

2. Verify it is up:

   ```bash
   curl http://localhost:3006/health
   ```

3. Connect from LangChain using
   [`langchain-mcp-adapters`](https://github.com/langchain-ai/langchain-mcp-adapters)
   over SSE (the SSE endpoint is `/sse`):

   ```python
   import asyncio
   from langchain_mcp_adapters.client import MultiServerMCPClient

   async def main():
       client = MultiServerMCPClient(
           {
               "transactions": {
                   "transport": "sse",
                   "url": "http://localhost:3006/sse",
               }
           }
       )
       tools = await client.get_tools()
       for tool in tools:
           print(tool.name, "-", tool.description)
       # Pass `tools` to your agent, e.g. langgraph's create_react_agent(model, tools)

   asyncio.run(main())
   ```

You can also run the HTTP server via Docker — see `docker-compose.yml` and the
`make docker-up` / `make compose-up` targets.

## Staging deploy (two stacks side by side)

To validate changes before promoting them, `mcp-api` can run a **staging**
instance alongside production. They are two independent compose stacks with
distinct container names and host ports, so both run at the same time:

| Stack | Compose file | Container | Host port |
| --- | --- | --- | --- |
| prod | `docker-compose.yml` | `mcp-api` | `6666` |
| staging | `docker-compose.staging.yml` | `mcp-api-staging` | `6667` |

Both stacks read a `stack.env` (gitignored — supplied per environment). Each
deployment provides its own values, so the **staging** `stack.env` sets
`SERVICE_CONTAINER_NAME=mcp-api-staging`, `SERVICE_PORT=6667` and a
`TRANSACTION_SERVICE_URL` pointing at the transactions backend. Both stacks join
the same external `financer-transactions_transactions-network` to reach it.

Local commands:

```bash
make staging-up      # build + start the staging stack (detached)
make staging-logs    # follow staging logs
make staging-down    # stop only staging (prod keeps running)
```

### Portainer

Create a **second stack** pointing at `docker-compose.staging.yml` and provide
its `stack.env` with the staging values (via the stack's *Environment variables*
panel or a mounted env file). The key requirement is that `SERVICE_CONTAINER_NAME`
differs from prod (`mcp-api-staging`) and `SERVICE_PORT` is a free host port
(`6667`) — that is what avoids the `container name "/mcp-api" is already in use`
conflict when deploying the second stack.

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `TRANSPORT` | `http` | `stdio` (Claude Desktop) or `http` (SSE for custom agents) |
| `SERVER_PORT` | `3006` | Port for HTTP/SSE mode |
| `TRANSACTION_SERVICE_URL` | `http://localhost:1235/api/v2` | Base URL of the backend transaction service |
| `LOG_LEVEL` | `info` | `info` or `debug` |
| `MCP_INSTRUCTIONS` | built-in | Overrides the instructions (server "system prompt") sent to clients |

## Development

```bash
make run     # go run .
make test    # go test ./...
make fmt     # gofmt + go mod tidy
make lint    # golangci-lint (if installed)
```

See [AGENTS.md](./AGENTS.md) for a deeper guide to the codebase.
