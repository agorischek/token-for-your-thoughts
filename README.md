# 🤖🗳️ Suggesting

Are your coding agents dealing with unreliable tools, conflicting instructions, and outdated documentation? Would you know it if they were?

__What if they could tell you?__

`suggesting` is utility for agents to provide feedback on tools, skills, etc. — so you can improve them. It can be used both via CLI and via MCP. Feedback can be sent to a file, an Open Telemetry endpoint, or any other process.

## Setup

First, install:

```bash
go install github.com/agorischek/suggesting/cmd/suggesting@latest
```

For MCP usage, add `suggesting serve-mcp` to your MCP config.

Update your `AGENTS.md` or similar instructions file to encourage agents to provide feedback.

Feedback will be written to `FEEDBACK.md` by default. See configuration docs for other sinks.

## Commands

Submit feedback directly:

```bash
suggesting submit \
  --provider "Claude Code" \
  --feedback "The command failed, but the surfaced output only showed a generic wrapper message, so I had to rerun it manually to see the real error."
```

Serve the MCP server:

```bash
suggesting serve-mcp
```

Print the version:

```bash
suggesting version
```

## Configuration

`suggesting` looks for `.suggesting.json` in the current directory and then walks up parent directories until it finds one. You can also pass `--config /path/to/.suggesting.json`.

If a `.env` file exists in the same directory as the resolved `.suggesting.json`, `suggesting` loads it automatically before resolving any `*_env` config fields. Existing process environment variables win over values from `.env`.

If `sinks` is omitted or empty, `suggesting` defaults to a single file sink that appends Markdown feedback to `FEEDBACK.md`.

Example:

```json
{
  "mcp": {
    "tool_name": "submit_feedback"
  },
  "sinks": [
    {
      "type": "file",
      "format": "markdown"
    }
  ]
}
```

### Sinks

#### `file`

- Appends Markdown entries to `FEEDBACK.md` by default.
- Supports `format: "markdown"` and `format: "json"`.
- JSON file output is newline-delimited JSON, so repeated submissions append cleanly. The default JSON filename is `feedback.jsonl`.
- Example:

```json
{
  "type": "file",
  "path": "FEEDBACK.md",
  "format": "markdown"
}
```

#### `git`

- Writes each feedback item into its own Markdown file named `{guid}.md`.
- Stores those files under a configurable directory such as `.feedback/`.
- Supports `format: "markdown"` and `format: "json"`.
- JSON git output writes one pretty-printed `{guid}.json` file per feedback item.
- Pushes to `origin` by default.
- Uses the current repository `HEAD` as the branch base if the feedback branch does not exist yet.
- Example:

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

- Spawns a configured subprocess.
- Sends one JSON-RPC 2.0 request over stdin and reads one response from stdout.
- The default method name is `submit_feedback`, but you can override it with `method`.
- The request `params` payload is the full feedback item: `id`, `provider`, `feedback`, `source`, `created_at`, and optional `metadata`.
- Example:

```json
{
  "type": "command",
  "command": "/usr/local/bin/feedback-bridge",
  "args": ["--stdio"],
  "method": "submit_feedback"
}
```

#### `application_insights`

- Sends each feedback item as a `customEvent` to the Application Insights ingestion endpoint.
- Uses a standard Application Insights connection string with `InstrumentationKey` and optionally `IngestionEndpoint` or `EndpointSuffix`.
- Supports either `connection_string` or `connection_string_env`, but not both.
- Defaults the event name to `suggesting feedback`, but you can override it with `event_name`.
- Example:

```json
{
  "type": "application_insights",
  "connection_string_env": "APPINSIGHTS_CONNECTION_STRING",
  "event_name": "suggesting feedback"
}
```

#### `sql`

- Does not bundle a database driver by default.
- Requires a driver name, DSN, and custom `insert_statement`.
- If you want `auto_create`, also provide a `create_statement`.
- Use a custom build that imports your SQL driver of choice.
- Example:

```json
{
  "type": "sql",
  "driver": "postgres",
  "dsn": "postgres://user:pass@localhost:5432/app?sslmode=disable",
  "insert_statement": "INSERT INTO feedback (id, provider, feedback, source, created_at, metadata_json) VALUES ($1, $2, $3, $4, $5, $6)"
}
```

#### `otel`

- Sends each feedback item as an OTLP log record.
- Uses the feedback text as the log body and attaches the ID, provider, source, timestamp, and metadata as attributes.
- Works with generic OTLP/HTTP log endpoints such as Better Stack when you point `endpoint` at `/v1/logs` and provide the required auth headers.
- Supports either `endpoint` or `endpoint_env`, and either `headers` or `headers_env`, but not both forms at once.
- `headers_env` should contain a JSON object string such as `{"Authorization":"Bearer ..."}`. In a `.env` file, wrap that JSON in single quotes.
- Example:

```json
{
  "type": "otel",
  "endpoint_env": "BETTER_STACK_OTEL_ENDPOINT",
  "headers_env": "BETTER_STACK_OTEL_HEADERS",
  "service_name": "suggesting"
}
```

## MCP Tool

The default MCP tool is named `submit_feedback`.

Default description:

> Submit feedback about your work context, including tool errors and inefficiencies, as well as information gaps and inconsistencies.

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
go build ./cmd/suggesting
```

Create a local snapshot release:

```bash
goreleaser release --snapshot --clean
```
