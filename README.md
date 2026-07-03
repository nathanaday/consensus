# Consensus

Analyzing time-series and IoT data usually means leaving the conversation:
export a CSV, open a notebook, remember the pandas incantation, write the SQL,
read the chart back. Consensus removes that round trip. It is a
[Model Context Protocol](https://modelcontextprotocol.io) server that lets an
MCP client — Claude or any other — load your time-series data and answer
questions about it in plain language. You point it at a source, and asking "what
was the trend last quarter?" becomes a tool call instead of a script.

The goal is to make a data stream something you can talk to.

## Status

Early and in active development. The server ingests time-series CSV files into a
local dataset store today; database sources (Postgres, SQLite) and the
statistical query tools that operate on stored datasets are the next phase.
There is also an `app/` surface reserved for a future full-stack front end — not
yet designed.

## Quickstart

Consensus is a Go MCP server that speaks over stdin/stdout. It is launched by an
MCP client, not run by hand.

### Prerequisites

- **Go 1.25+** — required by the MCP SDK. With the default `GOTOOLCHAIN=auto`,
  Go fetches the correct toolchain for you.

### Build

```bash
cd mcp
go build ./...
go test ./...
```

### Connect it to a client

Register the server with any MCP client. The command runs it over stdio:

```json
{
  "mcpServers": {
    "consensus": {
      "command": "go",
      "args": ["run", "./cmd/server"],
      "cwd": "/path/to/consensus/mcp"
    }
  }
}
```

Ingested datasets are written to `~/.consensus/store` by default; set
`CONSENSUS_STORE_DIR` to put them elsewhere.

## Usage

Ingestion is exposed as the `ingest_csv` tool. Give it a path; it detects the
timestamp column and the numeric value columns, normalizes the file to a
canonical layout, and stores it as Parquet.

```json
{ "path": "/data/Electric_Production_Small.csv" }
```

It returns a schema summary — never the raw rows — that later tools reference by
`dataset_id`:

```json
{
  "dataset_id": "electric_production_small",
  "series_ids": ["IPG2211A2N"],
  "row_count": 121,
  "time_range": { "start": "1985-01-01T00:00:00Z", "end": "1995-01-01T00:00:00Z" },
  "detected": { "timestamp_column": "DATE", "value_columns": ["IPG2211A2N"] }
}
```

A file with several numeric columns (say `humidity`, `smoke`, `temp`) becomes
one series per column under a single dataset. Override the detected columns with
`timestamp_col` and `value_cols` when a file's first row is ambiguous.

## How it works

Every source, whatever its shape, is normalized to one **long layout**:
`(timestamp, series_id, value)`. A wide CSV with N value columns becomes N
series sharing a timestamp axis. This gives every downstream tool a single
schema to reason about regardless of where the data came from.

Datasets are stored as Parquet files with a JSON catalog alongside them. The
catalog records schema and statistics — series ids, row counts, time ranges —
but never data values, so a client can survey what is available cheaply before
pulling anything.

## Repository layout

- `mcp/` — the Go MCP server; the only active surface today.
  - `internal/tools/` — one file per tool, each owning its input/output types.
  - `internal/ingest/` — CSV parsing and normalization to the long layout.
  - `internal/store/` — Parquet persistence and the dataset catalog.
- `app/` — reserved for a future full-stack application; stub only.
- `data/` — sample time-series CSVs for local testing.
- `docs/superpowers/` — design specs and implementation plans.

## Conventions

Full rationale in [`mcp/CONVENTIONS.md`](mcp/CONVENTIONS.md).

- **One tool per file** in `internal/tools/`, filename equal to the tool name.
- **A single registration seam.** `tools.Register` is the only place tools are
  wired into the server; the entry point never changes as tools are added.
- **The store knows nothing about sources.** Ingestion produces canonical rows;
  persistence just stores them, so new source types don't touch the store.
