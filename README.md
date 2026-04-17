# agent-eval

`agent-eval` is a local Go tool for evaluating MCP-based agent workflows through a deterministic task corpus.

## What it does

- serves an MCP evaluation surface over HTTP/SSE;
- runs task-by-task eval sessions through `eval_next` and `eval_submit`;
- stores sessions and answers in SQLite;
- exposes local history and CSV export workflows.

## Status

The repository is under active development. Some commands are already present in the CLI, while parts of the MCP server and reporting flow are still being implemented.

## Build

```bash
go build ./cmd/agent-eval
```

## Usage

```bash
./agent-eval serve
./agent-eval history
./agent-eval export
```

## Configuration

Runtime settings are loaded from `config.yaml`.

Current defaults include:

- `server.host`
- `server.port`
- `database.path`
- `session.idle_timeout_minutes`
- `tasks.randomize`
- `replication.enabled`
- `replication.count`
- `profiles`

## Data

- SQLite runtime state is stored locally.
- Task definitions live in `internal/tasks/tasks.yaml`.
