---
title: "Configuration"
description: "Environment variables, the Browse API credentials, defaults, and the data directory."
weight: 20
---

`ebay` needs almost no configuration: the core surfaces run anonymously against
public data out of the box. The settings below let you tune politeness, opt in
to the Browse API, and choose where data lands.

## Defaults

| Setting | Default | Flag |
|---|---|---|
| Requests | paced and retried on 429/5xx | `--rate`, `--retries` |
| Per-request timeout | 30s | `--timeout` |
| On-disk cache | under the data directory | `--no-cache` to bypass |

## Opting in to the Browse API

The item page and keyword search are soft-walled from datacenter IPs. When you
set both of these, those two commands fall back to eBay's documented Browse API:

```bash
export EBAY_CLIENT_ID=...        # your developer application's client id
export EBAY_CLIENT_SECRET=...    # its client secret
```

The credentials come from a free [eBay developer
application](https://developer.ebay.com/). `ebay` exchanges them for an
application token with the client-credentials grant, with no user login, and
reads them from the environment rather than flags so they never land in your
shell history. The reliable core surfaces never touch the API.

Set the marketplace the API answers for with `EBAY_MARKETPLACE` (for example
`EBAY_MARKETPLACE=EBAY_GB`); it defaults to the US marketplace.

## The data directory

Caches and any record store live under one data directory, chosen in this order:

1. `--data-dir`
2. `EBAY_DATA_DIR`
3. `$XDG_DATA_HOME/ebay`
4. `~/.local/share/ebay`

## Environment variables

Every flag has an environment fallback, prefixed `EBAY_` in upper case with
dashes as underscores. For example:

```bash
export EBAY_RATE=1s        # same as --rate 1s
export EBAY_DATA_DIR=~/data/ebay
```

Flags win over environment variables, which win over the built-in defaults.

## Sending records to a store

`--db` tees every emitted record into a store as a side effect of reading, so a
session fills a local database without a separate import step:

```bash
ebay category browse 9355 --db out.db        # SQLite file
ebay category browse 9355 --db 'postgres://...'
```
