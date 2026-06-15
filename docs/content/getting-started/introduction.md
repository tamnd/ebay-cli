---
title: "Introduction"
description: "What ebay is, how it is put together, and which surfaces it reads."
weight: 10
---

`ebay` reads public eBay data the way a logged-out browser does: a keyword
search, a single item, a seller and their listings, a category and the items in
it, the current deals, and search-box autocomplete. It is a single binary. It
speaks to eBay over plain HTTPS, shapes the responses into clean records, and
gets out of your way. There is no API key for the core, no login, and nothing to
run alongside it.

## How it is built

- A **library package** (`ebay`) holds the HTTP client and the typed data
  models. It paces requests, sends a browser User-Agent because that is what a
  logged-out reader looks like, caches on disk, and retries the transient
  failures any public site throws under load.
- A **domain** (`ebay/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference. It is the one place you add to the tool.
- A thin **`cmd/ebay`** hands the assembled app to `kit.Run`, which builds the
  command tree and the serve and mcp surfaces.

## One operation, four surfaces

Because an operation is surface-neutral, the same `seller show` you run on the
command line is also a route and a tool:

```bash
ebay seller show jomashop            # the command
ebay serve --addr :7777              # GET /v1/seller/show/jomashop
ebay mcp                             # the seller_show tool, over stdio
ant get ebay://seller/jomashop       # the URI dereference (via a host)
```

## What anonymous access reaches

eBay fronts its site with Akamai Bot Manager, and it does not treat every
surface the same. `ebay` sorts the surfaces into what it reads reliably and what
it cannot, and never pretends the line is elsewhere.

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
and the two walled commands fall back to it automatically. See
[configuration](/reference/configuration/#opting-in-to-the-browse-api).

Records carry only fields a logged-out reader can fill. There is no watch list,
no bids you placed, no offers, and no purchase history, because none of that
exists without an account. A storefront shows a seller's percent-positive,
lifetime items sold, and follower count, so those are the seller fields; the raw
feedback-score count is not on the storefront, so it is left empty rather than
guessed.

## Scope

`ebay` is a read-only client over data eBay already serves publicly. It reads
that data and shapes it for you. That narrow scope keeps it a single small
binary with no database, no daemon, and no setup.

`ebay` is an independent tool and is not affiliated with eBay.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
