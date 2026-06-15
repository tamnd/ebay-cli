---
title: "Quick start"
description: "Run your first ebay commands and shape their output."
weight: 30
---

Once `ebay` is on your `PATH`, complete a search box. `suggest` reads the
autocomplete host and answers from any network:

```bash
ebay suggest "mechanical keyboard" -n 6 --fields term
```

```
╭───────────────────────────────╮
│ TERM                          │
├───────────────────────────────┤
│ mechanical keyboard full size │
│ mechanical keyboard wireless  │
│ mechanical keyboard 75        │
│ mechanical keyboard 60        │
│ mechanical keyboard switches  │
│ mechanical keyboard vintage   │
╰───────────────────────────────╯
```

By default you get an aligned table on a terminal. Ask for JSON when you want to
pipe it:

```bash
ebay seller show jomashop -o json
```

```json
[
  {
    "username": "jomashop",
    "store": "Jomashop",
    "positive": 99.4,
    "sold": 768000,
    "followers": 78000,
    "url": "https://www.ebay.com/usr/jomashop"
  }
]
```

## Read the reliable surfaces

The category, seller, deals, and autocomplete surfaces read from any network:

```bash
ebay category browse 9355            # the items in a category
ebay category show 9355              # a category's metadata
ebay category tree 9355             # a category's child categories
ebay seller listings jomashop        # a seller's active listings
ebay deals                           # the current daily deals
```

The single item page and keyword search are best-effort and may hit eBay's bot
wall from a datacenter, exiting 4. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

```bash
ebay item 235791104766               # one item by id (best-effort)
ebay search iphone                   # keyword search (best-effort)
```

## Shape the output

The same flags work on every command:

```bash
ebay category browse 9355 --fields id,title,price,currency
ebay deals --template '{{.Title}} {{.Price}} {{.Currency}}'
ebay seller listings jomashop -o jsonl | jq .url
```

`-o` takes `table`, `json`, `jsonl`, `csv`, `tsv`, `url`, or `raw`. Left to
`auto`, it prints a table to a terminal and JSONL into a pipe, so the same
command reads well by hand and parses cleanly downstream. See
[output formats](/reference/output/) for the full contract.

## Resolve a reference offline

The `ref` commands classify and build eBay references with no network call:

```bash
ebay ref id "https://www.ebay.com/itm/Apple-iPhone-13/235791104766"
ebay ref url item 235791104766
```

## Serve it instead

The same operations are available over HTTP and to agents over MCP:

```bash
ebay serve --addr :7777 &
curl -s 'localhost:7777/v1/seller/show/jomashop'   # NDJSON, one record per line
ebay mcp                                            # MCP over stdio
```

## What to read next

The [guides](/guides/) cover the common jobs, and the
[CLI reference](/reference/cli/) is the full command tree and flag list.
