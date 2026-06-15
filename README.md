# ebay

A command line for [eBay](https://www.ebay.com). One binary that browses a
category, opens a seller and their listings, reads the daily deals, completes a
search box, resolves any eBay reference offline, and best-effort opens a single
item or runs a keyword search. No API key, no login, nothing to run alongside it.

```
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

On a terminal the table header and JSON values are colorized; piped to a file or
another program the output drops to plain text so it parses cleanly. Use
`--color always` to keep color through a pipe, or `--color never` to drop it.

Full documentation: [ebay-cli.tamnd.com](https://ebay-cli.tamnd.com).

## Why

Reading eBay programmatically usually means registering a developer application,
holding an OAuth token, and learning the Browse and Finding APIs, all to see
things a logged-out browser shows for free. `ebay` reads the same public pages a
visitor does, parses the schema.org data and the listing cards out of them, and
shapes each surface into a clean record with real output formats and pipelines
that compose.

eBay does not serve every surface to every network, and this tool is honest
about that. The category, seller, deals, and autocomplete surfaces answer from
any network. The single item page and keyword search sit behind eBay's bot
manager, which soft-walls them from datacenter IPs; those two are best-effort,
and they fall back to eBay's documented Browse API when you opt in with a free
application's credentials. See [what anonymous access reaches](#what-anonymous-access-reaches).

## Install

```sh
go install github.com/tamnd/ebay-cli/cmd/ebay@latest
```

Or grab a prebuilt binary from the [releases page](https://github.com/tamnd/ebay-cli/releases).
The binary is pure Go with no runtime dependencies. You can also run the
container image:

```sh
docker run --rm ghcr.io/tamnd/ebay:latest --help
```

Build from source:

```sh
git clone https://github.com/tamnd/ebay-cli
cd ebay-cli
make build      # produces ./bin/ebay
```

## Quick start

```sh
ebay suggest iphone                   # search-box autocomplete
ebay category browse 9355             # the items in a category
ebay category show 9355               # a category's metadata
ebay category tree 9355               # a category's child categories
ebay deals                            # the current daily deals
ebay seller show jomashop             # a seller's storefront
ebay seller listings jomashop         # a seller's active listings
ebay search iphone                    # keyword search (best-effort, see below)
ebay item 235791104766                # one item by id (best-effort, see below)
```

Most commands accept a bare id, a username, a `/itm/`, `/usr/`, `/str/`, or
`/b/` path, or a full eBay URL wherever they take a reference. The `ref`
commands resolve those offline, with no network call:

```sh
ebay ref id "https://www.ebay.com/itm/Apple-iPhone-13/235791104766" -o json
```

```json
[
  {
    "input": "https://www.ebay.com/itm/Apple-iPhone-13/235791104766",
    "kind": "item",
    "id": "235791104766",
    "url": "https://www.ebay.com/itm/235791104766"
  }
]
```

## How it works

eBay renders its public pages server-side, marking products up with schema.org
JSON-LD and listing cards in HTML. `ebay` GETs a page, parses the JSON-LD and
the cards, and maps each onto a clean record. The autocomplete host answers
plain JSON. It paces and caches requests and retries the transient failures, and
it sends a browser user-agent because that is what a logged-out reader looks
like. No API key, no token.

Prices are read in whatever currency eBay serves the page in, which depends on
the network's location, so each record carries an explicit `currency` field
alongside the number.

## What anonymous access reaches

eBay fronts its site with Akamai Bot Manager, and it does not treat every
surface the same. This tool sorts the surfaces into what it can read reliably and
what it cannot, and never pretends the line is elsewhere.

Read reliably from any network:

- `category browse`, `category show`, `category tree` (the `/b/` pages)
- `seller show`, `seller listings` (the `/usr/` and `/str/` storefronts)
- `deals` (the `/deals` hub)
- `suggest` (the autocomplete host)

Soft-walled from datacenter IPs, best-effort:

- `item` (the `/itm/` page)
- `search` (the `/sch/` keyword results)

From a home network these two usually answer; from a datacenter they often hit
the bot wall and exit 4. When that happens you have two remedies, and the error
message names both: run from a residential network, or opt in to eBay's
documented Browse API by setting `EBAY_CLIENT_ID` and `EBAY_CLIENT_SECRET` (a
free developer application, exchanged for an application token, no user login),
and the two walled commands fall back to it automatically.

Records carry only fields a logged-out reader can fill. There is no watch list,
no bids you placed, no offers, no purchase history, and no buyer-specific state,
because none of that exists without an account. A storefront shows a seller's
percent-positive, lifetime items sold, and follower count, so those are the
seller fields; the raw feedback-score count is not on the storefront, so it is
left empty rather than guessed.

When something is genuinely missing the exit code says which, so a script can
tell the cases apart:

| Exit | Meaning |
| --- | --- |
| 0 | ok |
| 2 | usage error |
| 3 | no results (the resource is genuinely empty) |
| 4 | need auth, or the bot wall (`item`, `search`) |
| 5 | rate limited (raise `--rate`) |
| 6 | not found (unknown id, removed listing, bad reference) |
| 8 | network error |

## Commands

| Command | What it does |
| --- | --- |
| `search <query>` | Keyword search (best-effort, see above) |
| `item <id>` | One item by id (best-effort, see above) |
| `deals` | The current daily deals |
| `suggest <prefix>` | Search-box autocomplete suggestions |
| `seller show <username>` | A seller's storefront profile |
| `seller listings <username>` | A seller's active listings |
| `category show <id>` | A category's metadata |
| `category browse <id>` | The items in a category |
| `category tree <id>` | A category's child categories |
| `ref id <ref>` | Classify a reference into its (kind, id), offline |
| `ref url <kind> <id>` | Build the canonical URL for a (kind, id), offline |
| `serve` | Serve the same operations over HTTP as NDJSON |
| `mcp` | Serve the same operations to an agent over MCP |
| `version` | Print version, commit, and build date |

A category is addressed by its numeric id, like `9355`, or a `/b/` URL. Run
`ebay <command> --help` for the full flag list on any command.

## Output

Every command shares one output contract. The default adapts to where output
goes, a table on a terminal and JSONL in a pipe, so the same command reads well
by hand and parses cleanly downstream.

```sh
ebay category browse 9355 -n 4 --fields id,title,condition
```

```
╭──────────────┬─────────────────────────────────────────────────────────────────────────────────┬─────────────────────────────────╮
│ ID           │ TITLE                                                                           │ CONDITION                       │
├──────────────┼─────────────────────────────────────────────────────────────────────────────────┼─────────────────────────────────┤
│ 146946310651 │ Apple iPhone 5C UNlocked 8/16/32GB BLUE/GREEN/PINK/WHITE/ YELLOW - New Battery  │ Pre-Owned · Apple               │
│ 377159482571 │ New S25 Ultra 5G 12+512GB Smartphone 6.9" Factory Unlocked Android Cellphones   │ Brand New · Unbranded           │
│ 376497333344 │ New Samsung Galaxy S22 Ultra 256GB/ 128GB 5G SM-S908U GSM CDMA Factory Unlocked │ Brand New · Samsung             │
│ 326282933769 │ Apple iPhone SE 3RD GEN 64GB Fully Unlocked 5G - VERY GOOD Condition            │ Very Good - Refurbished · Apple │
╰──────────────┴─────────────────────────────────────────────────────────────────────────────────┴─────────────────────────────────╯
```

Pick the format with `-o table|markdown|json|jsonl|csv|tsv|url|raw`, choose
columns with `--fields a,b,c`, render a custom line with `--template`, drop the
header with `--no-header`, and cap results with `-n/--limit`. The `url` format
prints just the canonical URL of each record, which is handy for piping into
another tool.

## Recipes

A seller's storefront as JSON, piped to jq:

```sh
ebay seller show jomashop -o json | jq '{store, positive, sold, followers}'
```

```json
{
  "store": "Jomashop",
  "positive": 99.4,
  "sold": 768000,
  "followers": 78000
}
```

Every item in a category as JSONL, for a downstream job:

```sh
ebay category browse 9355 -n 200 -o jsonl > phones.jsonl
```

The daily deals with the discount and both prices:

```sh
ebay deals -n 20 --fields title,price,was,currency,discount
```

The canonical URLs of a seller's listings, one per line:

```sh
ebay seller listings jomashop -n 50 -o url
```

Tee a category into a local SQLite store, keyed by each item's id, then query it:

```sh
ebay category browse 9355 -n 200 --db ebay.db
```

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```sh
ebay serve --addr :7777    # GET /v1/... returns NDJSON
ebay mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`ebay` registers an `ebay` domain the way a program registers a database driver
with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/ebay-cli/ebay"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `ebay://` URIs without knowing anything about eBay:

```sh
ant get ebay://item/<id>              # fetch an item
ant get ebay://seller/<username>      # fetch a seller
ant get ebay://category/<id>          # fetch a category
ant url ebay://category/<id>          # the live https URL
```

## Development

```
cmd/ebay/    thin main: hands cli.NewApp to kit.Run
cli/         assembles the kit App from the ebay domain
ebay/        the library: web client, Browse API backend, parsers, data
             models, and domain.go (the driver)
docs/        tago documentation site
```

```sh
make build      # ./bin/ebay
make test       # go test ./...
make vet        # go vet ./...
```

Every read command is declared once as a kit operation in `ebay/domain.go`. That
single declaration becomes the CLI subcommand, the HTTP route, and the MCP tool,
so the three surfaces never drift.

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the archives,
Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a cosign
signature:

```sh
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

`ebay` is an independent tool and is not affiliated with eBay. Apache-2.0, see
[LICENSE](LICENSE).
