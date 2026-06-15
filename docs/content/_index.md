---
title: "ebay"
description: "ebay reads public eBay data (categories, sellers, deals, autocomplete, and best-effort items and search) into structured records over a CLI, an HTTP server, and an MCP tool set."
heroTitle: "ebay, from the command line"
heroLead: "Browse a category, open a seller and their listings, read the daily deals, complete a search box, and best-effort open a single item or run a search. One pure-Go binary, no API key, output that pipes into the rest of your tools, and a resource-URI driver other programs can address."
heroPrimaryURL: "/getting-started/quick-start/"
heroPrimaryText: "Get started"
---

`ebay` reads the public eBay pages a logged-out browser sees, shapes them into
clean records, and gets out of your way.

```bash
ebay suggest iphone           # search-box autocomplete
ebay category browse 9355     # the items in a category
ebay seller show jomashop     # a seller's storefront
ebay deals                    # the current daily deals
ebay serve --addr :7777       # the same operations over HTTP
```

There is no API key for the core, no login, and nothing to run alongside it.
Output adapts to where it goes: an aligned table on your terminal, JSONL the
moment you pipe it somewhere.

## Honest about what is reachable

eBay fronts its site with a bot manager that does not treat every surface the
same. `ebay` is explicit about the line. The category, seller, deals, and
autocomplete surfaces read from any network. The single item page and keyword
search are soft-walled from datacenter IPs, so they are best-effort and fall
back to eBay's documented Browse API when you opt in with a free application's
credentials. See [what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

## Two ways to use it

- **As a command** for reading eBay by hand or in a script. Start with
  the [quick start](/getting-started/quick-start/).
- **As a resource-URI driver** so a host like
  [ant](https://github.com/tamnd/ant) can address eBay as `ebay://` URIs and
  follow links across sites. See [resource URIs](/guides/resource-uris/).

Both are the same code: one operation, declared once, is a CLI command, an HTTP
route, an MCP tool, and a URI dereference.

## Where to go next

- New here? Read the [introduction](/getting-started/introduction/), then the
  [quick start](/getting-started/quick-start/).
- Installing? See [installation](/getting-started/installation/).
- Doing a specific job? The [guides](/guides/) are task-first.
- Need every flag? The [CLI reference](/reference/cli/) is the full surface.
