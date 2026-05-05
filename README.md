# suggestion-box

`suggestion-box` is a Go CLI for collecting feedback from coding agents about their experience working in a repository. It supports direct submission from the command line and an MCP server mode that exposes a `submit_feedback` tool by default.

Each feedback record gets a GUID and must include a `Provider`, such as `Claude Code`, so teams can trace where the report came from. The default MCP tool description is tuned for reporting tool errors and inefficiencies, along with instruction gaps and inconsistencies. Both the tool name and description can be overridden in `.suggestionsrc`.

## Features

- `submit` command for direct feedback submission
- `serve-mcp` command for exposing an MCP tool over stdio
- JSON configuration via `.suggestionsrc`
- Multiple sinks per submission
- Built-in sinks for OTel, file, Git, and SQL
- GoReleaser config and release workflow

## Installation

```bash
go install github.com/agorischek/suggestion-box/cmd/suggestion-box@latest
```

## Commands

Submit feedback directly:

```bash
suggestion-box submit \
  --provider "Claude Code" \
  --summary "Shell output hid the real error" \
  --details "The command failed, but the surfaced output only showed a generic wrapper message." \
  --category tooling \
  --metadata '{"command":"go test ./...","exit_code":1}'
```

Serve the MCP server:

```bash
suggestion-box serve-mcp
```

Print the version:

```bash
suggestion-box version
```

## Configuration

`suggestion-box` looks for `.suggestionsrc` in the current directory and then walks up parent directories until it finds one. You can also pass `--config /path/to/.suggestionsrc`.

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
      "branch": "agent-feedback",
      "remote": "origin",
      "path": "FEEDBACK.md"
    },
    {
      "type": "sql",
      "driver": "sqlite",
      "dsn": "feedback.db",
      "table": "feedback"
    },
    {
      "type": "otel",
      "endpoint": "http://localhost:4318",
      "insecure": true,
      "service_name": "suggestion-box"
    }
  ]
}
```

### Sink Notes

`file`

- Appends Markdown entries to `FEEDBACK.md` by default.

`git`

- Writes feedback into a dedicated branch and commits a new entry there.
- Pushes to `origin` by default.
- Uses the current repository `HEAD` as the branch base if the feedback branch does not exist yet.

`sql`

- Supports SQLite out of the box through `modernc.org/sqlite`.
- For non-SQLite drivers, provide a custom `insert_statement`.

`otel`

- Sends each feedback item as an OTLP trace span with feedback attributes attached.

## MCP Tool

The default MCP tool is named `submit_feedback`.

Default description:

> Submit feedback about working in this repository, including tool errors and inefficiencies, as well as instruction gaps and inconsistencies.

Tool input fields:

- `provider` required
- `summary` required
- `details` optional
- `category` optional
- `metadata` optional JSON object

## Development

Run tests:

```bash
go test ./...
```

Build the CLI:

```bash
go build ./cmd/suggestion-box
```

Create a local snapshot release:

```bash
goreleaser release --snapshot --clean
```
