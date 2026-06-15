---
title: "Output formats"
description: "The output contract every command shares: formats, fields, and templates."
weight: 30
---

Every command renders through one formatter, so the same flags work everywhere.
Pick a format with `-o`, or let `ebay` choose: a table when writing to a
terminal, JSONL when piped.

## Formats

```bash
ebay category browse 9355 -o table   # aligned columns for reading
ebay category browse 9355 -o jsonl   # one JSON object per line, for piping
ebay category browse 9355 -o json    # a single JSON array
ebay category browse 9355 -o csv     # spreadsheet friendly
ebay category browse 9355 -o tsv     # tab-separated
ebay category browse 9355 -o url     # just the URL column
ebay category browse 9355 -o raw     # the underlying bytes, unformatted
```

| Format | Best for |
|---|---|
| `table` | Reading on a terminal |
| `jsonl` | Piping into another tool, one object at a time |
| `json` | Loading a whole result as an array |
| `csv` / `tsv` | Spreadsheets and quick column math |
| `url` | Feeding URLs into other commands |
| `raw` | The unformatted bytes (response bodies) |

## Narrowing columns

Keep only the fields you want:

```bash
ebay category browse 9355 --fields id,title,price,currency
```

`--no-header` drops the header row in `table` and `csv` output, which helps when
a downstream tool expects bare rows.

## Templating rows

For full control over each line, apply a Go text/template. Fields are the JSON
keys, capitalised:

```bash
ebay deals --template '{{.Title}} {{.Price}} {{.Currency}}'
```

## Prices carry their currency

eBay serves prices in the currency tied to the network's location, so every
listing, deal, and item record carries an explicit `currency` field alongside
the number. Keep the two together when you template or filter, since the same
field can read in USD, GBP, or VND depending on where you run from.

## Why auto-detection helps

Because the default adapts to the destination, the same command reads well by
hand and parses cleanly in a pipe:

```bash
ebay seller listings jomashop            # a table, because this is a terminal
ebay seller listings jomashop | wc -l    # JSONL, because this is a pipe
```

You only reach for `-o` when you want something other than that default.
