<img src="./assets/logo.png" alt="Token For Your Thoughts logo" height="150" />

# Token For Your Thoughts

Are your coding agents dealing with unreliable tools, conflicting instructions, and outdated documentation? Would you know it if they were? _What if they could tell you?_

Token For Your Thoughts (`tfyt`) is a utility for agents to provide feedback on tools, skills, and overall repository experience — so you can make things better for them. It can be used via both MCP and CLI, and feedback can be sent to a file, an HTTP webhook, an OpenTelemetry endpoint, or any other process.

## Setup

You can install `tfyt` either by downloading a prebuilt binary from the [GitHub Releases page](https://github.com/agorischek/token-for-your-thoughts/releases) or by building and installing it with Go.

To install with Go:

```bash
go install github.com/agorischek/token-for-your-thoughts/cmd/tfyt@latest
```

For MCP usage, add `tfyt serve-mcp` to your MCP config.

Update your `AGENTS.md` or similar instructions file to encourage agents to provide feedback.

Feedback will be written to `FEEDBACK.md` by default. See below for configuration of other destinations.

## Commands

Submit feedback directly:

```bash
tfyt submit \
  --provider "Claude Code" \
  --feedback "The command failed, but the surfaced output only showed a generic wrapper message, so I had to rerun it manually to see the real error." \
  --metadata '{"command":"git status","exit_code":1}'
```

Serve the MCP server:

```bash
tfyt serve-mcp
```

Print the version:

```bash
tfyt version
```

Update to the latest release:

```bash
tfyt update
```

Set `GITHUB_TOKEN` for authenticated GitHub API access (avoids rate limits):

```bash
export GITHUB_TOKEN="$(gh auth token)"
tfyt update
```

## Configuration

`tfyt` looks for `.tfyt.toml` first and then `.tfyt.json` in the current directory, walking up parent directories until it finds one. You can also pass `--config /path/to/.tfyt.toml` or `--config /path/to/.tfyt.json`. If no config file is found, `tfyt` falls back to writing feedback to `FEEDBACK.md` in the current directory.

If you set `env_file_path`, `tfyt` loads that env file before resolving any `*_env` config fields. Relative paths are resolved relative to the config file. Existing process environment variables win over values from the env file.

A JSON Schema for the config lives at [tfyt.schema.json](/Users/umeboshi/Git/token-for-your-thoughts/tfyt.schema.json) and is included in GitHub release archives together with the example configs.

If `destinations` is omitted or empty, `tfyt` defaults to a single file destination that appends Markdown feedback to `FEEDBACK.md`.

Example:

```json
{
  "mcp": {
    "tool_name": "submit_feedback"
  },
  "destinations": [
    {
      "type": "file",
      "format": "markdown"
    }
  ]
}
```

### Destinations

#### `file`

The file destination is the default and is the simplest way to start collecting feedback. In Markdown mode it appends entries to `FEEDBACK.md`. In JSON mode it writes newline-delimited JSON so repeated submissions can be appended safely without rewriting the whole file; the default JSON filename is `feedback.jsonl`.

```json
{
  "type": "file",
  "path": "FEEDBACK.md",
  "format": "markdown"
}
```

#### `git`

The git destination writes each feedback item into its own file on a dedicated branch, which is useful when you want feedback tracked alongside the repository without mixing it into the main working branch. Files are stored under a configurable directory such as `.feedback/`, with one `{guid}.md` or `{guid}.json` file per submission depending on the selected format. By default it pushes to `origin`, and if the feedback branch does not exist yet it uses the current repository `HEAD` as the starting point.

```json
{
  "type": "git",
  "branch": "feedback",
  "remote": "origin",
  "directory": ".feedback",
  "format": "markdown"
}
```

#### `command`

The command destination hands feedback off to another local process over JSON-RPC 2.0. `tfyt` starts the configured subprocess, sends the full feedback item on stdin, and reads the response from stdout. The default JSON-RPC method name is `submit_feedback`, though you can override it with `method`. When `tfyt` is running as an MCP server, the subprocess is kept alive and reused across submissions instead of being restarted every time.

```json
{
  "type": "command",
  "command": "/usr/local/bin/feedback-bridge",
  "args": ["--stdio"],
  "method": "submit_feedback"
}
```

#### `http`

The HTTP destination sends each feedback item as JSON to a webhook or ingestion endpoint. It uses `POST` by default, but you can choose a different method if your endpoint expects one. You can configure the URL and headers directly or source them from environment variables with `url_env` and `headers_env`, but each setting must come from only one place. Requests default to a `10` second timeout, and if you do not specify `success_statuses`, any `2xx` response is treated as success. When using `headers_env`, the env var should contain a JSON object string such as `{"Authorization":"Bearer ..."}`; in a `.env` file, wrap that JSON in single quotes.

```json
{
  "type": "http",
  "url_env": "TFYT_HTTP_URL",
  "headers_env": "TFYT_HTTP_HEADERS",
  "timeout_seconds": 10,
  "success_statuses": [202]
}
```

#### `application_insights`

The Application Insights destination sends each feedback item as a `customEvent` to Azure Application Insights. It uses a standard connection string with `InstrumentationKey` and can also honor `IngestionEndpoint` or `EndpointSuffix` when those are present. You can provide the connection string directly or through `connection_string_env`, but not both at once. The event name defaults to `tfyt feedback`, and you can change it with `event_name` if you want the events grouped differently in Azure.

```json
{
  "type": "application_insights",
  "connection_string_env": "APPINSIGHTS_CONNECTION_STRING",
  "event_name": "tfyt feedback"
}
```

#### `sql`

The SQL destination is intentionally minimal and expects you to bring your own driver. You provide the driver name, DSN, and an `insert_statement`, which lets the destination work with many SQL backends without baking one specific database into the binary. If you want `tfyt` to create the table automatically, also provide a `create_statement`. In practice this destination is best used in a custom build that imports the SQL driver you want.

The `insert_statement` receives exactly six positional parameters in this order:

1. `id` — feedback GUID (string)
2. `provider` — provider name (string)
3. `feedback` — feedback text (string)
4. `source` — submission source, e.g. `cli` or `mcp` (string)
5. `created_at` — RFC 3339 timestamp (string)
6. `metadata_json` — metadata as a JSON object (string)

```json
{
  "type": "sql",
  "driver": "postgres",
  "dsn": "postgres://user:pass@localhost:5432/app?sslmode=disable",
  "insert_statement": "INSERT INTO feedback (id, provider, feedback, source, created_at, metadata_json) VALUES ($1, $2, $3, $4, $5, $6)"
}
```

#### `otel`

The OpenTelemetry destination emits each feedback item as an OTLP log record. The feedback text becomes the log body, while the ID, provider, source, timestamp, and metadata are attached as attributes. This works well with generic OTLP/HTTP log backends such as Better Stack when you point `endpoint` at a `/v1/logs` URL and provide the necessary auth headers. As with the HTTP destination, `endpoint` and `headers` can come either directly from config or from `*_env` fields, but not both at the same time; if you use `headers_env`, the env var should contain a JSON object string and should be single-quoted in a `.env` file.

```json
{
  "type": "otel",
  "endpoint_env": "BETTER_STACK_OTEL_ENDPOINT",
  "headers_env": "BETTER_STACK_OTEL_HEADERS",
  "service_name": "tfyt"
}
```

## MCP Tool

The default MCP tool is named `submit_feedback`. Both the tool name and description can be overriden via configuration.

Tool input fields:

- `provider` required
- `feedback` required
- `metadata` optional JSON object

## Development

Run tests:

```bash
go test ./...
```

Build the CLI:

```bash
go build ./cmd/tfyt
```

Create a local snapshot release:

```bash
goreleaser release --snapshot --clean
```
