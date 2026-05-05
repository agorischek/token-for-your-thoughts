# suggesting

`suggesting` is a Go CLI for collecting feedback from coding agents about their experience working in a repository. It supports direct submission from the command line and an MCP server mode that exposes a `submit_feedback` tool by default.

Each feedback record gets a GUID and must include a `Provider`, such as `Claude Code`, so teams can trace where the report came from. The feedback itself is a single text field rather than separate summary/detail fields. The default MCP tool description is tuned for reporting tool errors and inefficiencies, along with instruction gaps and inconsistencies. Both the tool name and description can be overridden in `.suggesting.json`.

## Features

- `submit` command for direct feedback submission
- `serve-mcp` command for exposing an MCP tool over stdio
- JSON configuration via `.suggesting.json`
- Multiple sinks per submission
- Built-in sinks for OTel, file, Git, command, Application Insights, and SQL
- GoReleaser config and release workflow

## Installation

```bash
go install github.com/agorischek/suggesting/cmd/suggesting@latest
```

## Commands

Submit feedback directly:

```bash
suggesting submit \
  --provider "Claude Code" \
  --feedback "The command failed, but the surfaced output only showed a generic wrapper message, so I had to rerun it manually to see the real error." \
  --metadata '{"command":"go test ./...","exit_code":1}'
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

Example:

```json
{
  "mcp": {
    "tool_name": "submit_feedback",
    "tool_description": "Submit feedback about working in this repository, including tool errors and inefficiencies, as well as instruction gaps and inconsistencies."
  },
  "sinks": [
    {
      "type": "file",
      "path": "FEEDBACK.md"
    },
    {
      "type": "git",
      "branch": "feedback",
      "remote": "origin",
      "directory": ".feedback"
    },
    {
      "type": "command",
      "command": "/usr/local/bin/feedback-bridge",
      "args": ["--stdio"],
      "method": "submit_feedback"
    },
    {
      "type": "application_insights",
      "connection_string": "InstrumentationKey=00000000-0000-0000-0000-000000000000;IngestionEndpoint=https://dc.applicationinsights.azure.com/",
      "event_name": "suggesting feedback"
    },
    {
      "type": "sql",
      "driver": "postgres",
      "dsn": "postgres://user:pass@localhost:5432/app?sslmode=disable",
      "insert_statement": "INSERT INTO feedback (id, provider, feedback, source, created_at, metadata_json) VALUES ($1, $2, $3, $4, $5, $6)"
    },
    {
      "type": "otel",
      "endpoint": "http://localhost:4318",
      "insecure": true,
      "service_name": "suggesting"
    }
  ]
}
```

### Sink Notes

`file`

- Appends Markdown entries to `FEEDBACK.md` by default.

`git`

- Writes each feedback item into its own Markdown file named `{guid}.md`.
- Stores those files under a configurable directory such as `.feedback/`.
- Pushes to `origin` by default.
- Uses the current repository `HEAD` as the branch base if the feedback branch does not exist yet.

`command`

- Spawns a configured subprocess.
- Sends one JSON-RPC 2.0 request over stdin and reads one response from stdout.
- The default method name is `submit_feedback`, but you can override it with `method`.
- The request `params` payload is the full feedback item: `id`, `provider`, `feedback`, `source`, `created_at`, and optional `metadata`.

`application_insights`

- Sends each feedback item as a `customEvent` to the Application Insights ingestion endpoint.
- Uses a standard Application Insights connection string with `InstrumentationKey` and optionally `IngestionEndpoint` or `EndpointSuffix`.
- Defaults the event name to `suggesting feedback`, but you can override it with `event_name`.

`sql`

- Does not bundle a database driver by default.
- Requires a driver name, DSN, and custom `insert_statement`.
- If you want `auto_create`, also provide a `create_statement`.
- Use a custom build that imports your SQL driver of choice.

`otel`

- Sends each feedback item as an OTLP trace span with feedback attributes attached.

## MCP Tool

The default MCP tool is named `submit_feedback`.

Default description:

> Submit feedback about working in this repository, including tool errors and inefficiencies, as well as instruction gaps and inconsistencies.

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
