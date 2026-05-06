# Agent Guide

This guide is for coding agents that are helping a user install, configure, or integrate `tfyt` in another repository. The goal is not just to explain what `tfyt` can do, but to help you make reasonable contextual decisions so you can present the user with a small set of good options instead of a long questionnaire.

## Goal

Use `tfyt` to make it easier for agents working in a repository to report friction, breakage, confusing instructions, and workflow inefficiencies. In most repos, the right outcome is:

- `tfyt` is installed in a way that matches the repo’s existing tooling norms.
- agents can call `submit_feedback` through MCP, CLI, or both.
- feedback goes somewhere the humans maintaining the repo will actually look.

## Working Style

When you are helping a user adopt `tfyt`, prefer to inspect the repository first and then present a few concrete recommendations. Do not ask the user to design the whole setup from scratch if the codebase already suggests a sensible path.

Good signals to look for:

- whether the repo is Go-based
- whether the repo already manages pinned tools via a lockfile or tool manifest
- whether there is already an MCP configuration file
- whether the repo already uses OpenTelemetry, Better Stack, Application Insights, GitHub Discussions, or database-backed operational tooling
- whether the repo uses a checked-in `AGENTS.md`, `CLAUDE.md`, `.github/copilot-instructions.md`, or similar agent instructions
- whether a checked-in `FEEDBACK.md` or `.feedback/` workflow would fit the repo culture

After you inspect the repo, present the user with recommended options and briefly explain why each one fits.

## Installation

Choose the installation path that best matches the target repository.

### Preferred Order

1. If the repo already installs Go tools through a pinned manifest or lockfile, add `tfyt` there.
2. Else, if the repo is a Go project but does not pin tools that way, use `go install`.
3. Else, use a GitHub Releases binary and place it somewhere on the user’s `PATH`, usually a user-local bin directory.

### What To Look For

Signs that the repo already manages Go tools in a pinned way:

- a `tools.go` file
- `go tool` wrappers or a documented tool bootstrap flow
- `mise`, `aqua`, `asdf`, `devbox`, `nix`, or similar tool manifests
- `Taskfile`, `Makefile`, or bootstrap scripts that install specific tool versions

In that case, follow the existing pattern rather than inventing a new one.

If it is a normal Go repo without pinned tool management, `go install github.com/agorischek/token-for-your-thoughts/cmd/tfyt@latest` is usually the right answer.

If it is not primarily a Go repo, prefer downloading the correct binary from GitHub Releases and placing it in a user bin directory such as `~/.local/bin`, `~/bin`, or another repo-standard tools directory. Choose the asset that matches the user’s OS and architecture.

### Versioning Guidance

If the repo cares about reproducibility, prefer a pinned version rather than `@latest`. If the repo already has a release management pattern, match it.

## Configuration Files

`tfyt` looks for `.tfyt.toml` first and then `.tfyt.json`. Prefer TOML unless the target repo strongly prefers JSON configuration.

Use `env_file_path` only when the repo needs a checked-in config that refers to secrets stored in an env file. If the repo already has a standard secrets mechanism, follow that instead.

Remember the precedence rule:

- if both `x` and `x_env` are set
- and `x_env` is nonempty
- then `x_env` wins
- and if the env var is missing or invalid, loading fails rather than falling back to `x`

## MCP Setup

If the repo already uses MCP, try to integrate `tfyt serve-mcp` into the existing MCP config rather than introducing a parallel mechanism. Look for files such as:

- `.mcp.json`
- `.vscode/mcp.json`
- tool-specific MCP config files

If there is no MCP setup yet, recommend one only if the repository is clearly using agent tools that benefit from MCP. Otherwise, a direct CLI setup may be enough.

When you add MCP, also update agent-facing instructions so agents are explicitly encouraged to use `submit_feedback`.

## Choosing Destinations

Do not ask the user to choose from every destination immediately. Instead, inspect the repo and propose the most plausible one to three options.

### Good Default

For many repos, start with:

- `file` writing to `FEEDBACK.md`

This is low-friction, easy to review in pull requests, and requires no external services.

### When To Prefer `git`

Recommend `git` when:

- the team wants one feedback file per submission
- a dedicated feedback branch fits existing repo workflows
- maintainers are comfortable reviewing generated files on a side branch

### When To Prefer `github_discussions`

Recommend `github_discussions` when:

- the repo already uses GitHub Discussions as an inbox or triage surface
- maintainers want feedback to show up in GitHub rather than in the working tree
- the user already has or can provide a GitHub token

If you recommend it, suggest a likely category based on existing discussion categories, and mention `GITHUB_TOKEN` if that is the easiest credential source.

### When To Prefer `http`

Recommend `http` when:

- the repo already has an internal webhook, ingest endpoint, or automation service
- there is a simple external system that just needs JSON POSTs

### When To Prefer `otel`

Recommend `otel` when:

- the repo already uses OpenTelemetry
- the team already ships telemetry to a backend such as Better Stack
- maintainers want feedback in their existing observability pipeline

### When To Prefer `application_insights`

Recommend `application_insights` when:

- the repo or organization already uses Azure Application Insights
- the maintainers think of feedback as operational telemetry

### When To Prefer `sql`

Recommend `sql` only when:

- there is already a database-backed workflow for collecting internal operational data
- the team is comfortable providing schema and insert statements

### When To Prefer `command`

Recommend `command` when:

- the repo already has a script, hook, or lightweight executable that should run once per feedback item
- the integration is local and simple

Explain the two payload modes:

- `content_mode = "json"` sends one JSON payload on stdin
- `content_mode = "args"` sends structured CLI flags

### When To Prefer `process`

Recommend `process` when:

- the destination is a long-lived helper process
- JSON-RPC over stdio is already a natural fit
- the repo benefits from keeping the subprocess warm, especially in MCP usage

## Presenting Options To The User

After inspecting the repo, present a short recommendation set. A good answer shape is:

1. one recommended default
2. one or two alternatives
3. one sentence on why each option fits this repo

For example:

1. `file` to `FEEDBACK.md`
   Best when the repo is small, local-first, and already uses checked-in docs as process memory.
2. `github_discussions`
   Best when the team already triages work in GitHub and wants feedback out of the working tree.
3. `otel`
   Best when the repo already has Better Stack or OTLP infrastructure in place.

Prefer making a recommendation over asking an open-ended “which destination do you want?” question.

## Agent Instructions

If the target repo has agent instruction files, update them so agents know:

- when to use `submit_feedback`
- what kinds of issues are worth reporting
- that they should include their provider name
- that concrete feedback and metadata are more useful than vague complaints

Good places to update include:

- `AGENTS.md`
- `.github/copilot-instructions.md`
- `CLAUDE.md`
- repo onboarding docs for local agents

## Suggested Minimal Rollouts

If the repo is brand new to `tfyt`, prefer one of these rollouts:

### Local-first rollout

- install `tfyt`
- add `.tfyt.toml`
- configure `file` to `FEEDBACK.md`
- add MCP if the repo already uses agent tooling
- update `AGENTS.md`

### GitHub-native rollout

- install `tfyt`
- configure `github_discussions`
- add MCP
- update agent instructions to report friction into Discussions

### Observability rollout

- install `tfyt`
- configure `otel` or `application_insights`
- store secrets in env vars or env file
- add MCP and direct CLI examples

## Final Checks

Before you finish, verify:

- the chosen install path matches the repo’s existing conventions
- the config file is valid
- any `*_env` references resolve correctly
- the MCP server starts if you configured MCP
- at least one real feedback submission succeeds
- the destination actually receives the feedback

If you are unsure between two approaches, recommend one and explain the tradeoff rather than pushing the decision back to the user without context.
