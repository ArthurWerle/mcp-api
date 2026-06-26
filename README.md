# mcp-api

An [MCP](https://modelcontextprotocol.io/) server that gives an LLM read-only
access to a personal-finance **transactions** dataset. It acts as a thin bridge
between an MCP client (Claude Desktop, a custom agent, etc.) and a backend
transaction REST service, exposing a handful of finance-focused tools.

## Tools

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
| `health_check` | Check the backend transactions service |

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
