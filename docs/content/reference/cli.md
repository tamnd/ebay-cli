---
title: "CLI"
description: "Every command and subcommand, with the flags that matter."
weight: 10
---

```
ebay <command> [arguments] [flags]
```

Run `ebay <command> --help` for the full flag list on any command.

## Commands

| Command | What it does |
|---|---|
| `search <query>` | Keyword search (best-effort, may hit the bot wall) |
| `item <id>` | Show one item by id (best-effort, may hit the bot wall) |
| `deals` | The current daily deals |
| `suggest <prefix>` | Search-box autocomplete suggestions |
| `seller show <username>` | Show a seller's storefront profile |
| `seller listings <username>` | List a seller's active listings |
| `category show <id>` | Show a category's metadata |
| `category browse <id>` | List the items in a category |
| `category tree <id>` | List a category's child categories |
| `ref id <ref>` | Classify a reference into its (kind, id), offline |
| `ref url <kind> <id>` | Build the canonical URL for a (kind, id), offline |
| `serve [--addr]` | Serve the operations over HTTP as NDJSON |
| `mcp` | Run as an MCP server over stdio |
| `version` | Print the version and exit |

A category is addressed by its numeric id, like `9355`, or a `/b/` URL. A seller
is addressed by username, store slug, or URL. The `item` and `search` commands
are soft-walled from datacenter IPs; see
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

## Global flags

These are shared by every operation, so they work the same on every command.

| Flag | Meaning |
|---|---|
| `-o, --output` | Output format: `auto`, `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, `raw` |
| `--fields` | Comma-separated columns to keep |
| `--template` | Go text/template applied per record |
| `--no-header` | Omit the header row in `table` and `csv` |
| `-n, --limit` | Stop after N records (0 means no limit) |
| `--rate` | Minimum delay between requests |
| `--retries` | Retry attempts on rate limit or 5xx |
| `--timeout` | Per-request timeout |
| `--data-dir` | Override the data directory |
| `--no-cache` | Bypass on-disk caches |
| `--db` | Tee every record into a store (e.g. `out.db`, `postgres://...`) |
| `-v, --verbose` | Increase verbosity (repeatable) |
| `-q, --quiet` | Suppress progress output |
| `--color` | `auto`, `always`, or `never` |
| `--user-agent` | Override the User-Agent sent with each request |
| `--marketplace` | Browse API marketplace id (default `EBAY_US`) |
| `--cache-ttl` | How long a cached response stays fresh |
| `--refresh` | Fetch fresh copies and rewrite the cache, ignoring any hit |

See [output formats](/reference/output/) for what `-o`, `--fields`, and
`--template` produce, and [configuration](/reference/configuration/) for
environment variables, the Browse API credentials, and defaults.
