---
title: "Resource URIs"
description: "Use ebay as a database/sql-style driver so a host program can address eBay as ebay:// URIs."
weight: 20
---

`ebay` is a command line, but the `ebay` Go package is also a small driver that
makes eBay addressable as a resource URI. A host program registers it the way a
program registers a database driver with `database/sql`, then dereferences
`ebay://` URIs without knowing anything about how eBay is fetched.

The host that does this today is [ant](https://github.com/tamnd/ant), a single
binary that puts one URI namespace over a family of site tools. The examples
below use `ant`; any program that links the package gets the same behaviour.

## Mounting the driver

A host enables the driver with one blank import, exactly like
`import _ "github.com/lib/pq"`:

```go
import _ "github.com/tamnd/ebay-cli/ebay"
```

The package's `init` registers a domain with the scheme `ebay` for the hosts
`www.ebay.com`, `ebay.com`, and `m.ebay.com`. The standalone `ebay` binary does
not change.

## Addressing records

A URI is `scheme://authority/id`. The resolver types are:

| URI                        | What it is                          |
| -------------------------- | ----------------------------------- |
| `ebay://item/<id>`         | one item, keyed by its item id      |
| `ebay://seller/<username>` | a seller's storefront profile       |
| `ebay://category/<id>`     | a category, keyed by its numeric id |

```bash
ant get ebay://seller/jomashop          # the seller record
ant get ebay://category/9355            # the category record
ant url ebay://item/235791104766        # the live https URL
ant resolve https://www.ebay.com/usr/jomashop  # a pasted link, back to its URI
```

`item` is best-effort: from a datacenter it may hit eBay's bot wall and report
need-auth, the same as the `ebay item` command. See
[what anonymous access reaches](/getting-started/introduction/#what-anonymous-access-reaches).

## Walking the graph

`ls` lists the members of a collection, and every member is itself an
addressable URI, so a host can follow the graph and write it to disk:

```bash
ant ls     ebay://category/9355           # the items in the category, as item URIs
ant export ebay://seller/jomashop --follow 1 --to ./data
```

The list operations (`category browse`, `category tree`, `seller listings`,
`deals`, `search`) emit records that are themselves addressable, so each member
is an `ebay://item/` or `ebay://category/` URI a host can fetch in turn.

## Why this is the same code

The driver and the binary share one definition per operation. A resolver op
answers both `ebay seller show` on the command line and
`ant get ebay://seller/...` through a host, from the same handler and the same
client. There is no second implementation to keep in step.
