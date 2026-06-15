---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how eBay serves its data, not a
bug. Each case maps to an exit code so a script can tell them apart.

## `item` or `search` exits 4

These two surfaces sit behind eBay's bot manager, which soft-walls them from
datacenter IPs. From a home network they usually answer; from a datacenter or a
cloud host they often hit the wall, and `ebay` reports need-auth (exit 4) rather
than pretending the result was empty. Two remedies, and the error message names
both: run from a residential network, or opt in to the Browse API by setting
`EBAY_CLIENT_ID` and `EBAY_CLIENT_SECRET` so the two commands fall back to it.
See [configuration](/reference/configuration/#opting-in-to-the-browse-api).

The category, seller, deals, and suggest surfaces are not affected; they read
from any network.

## Requests start failing or returning 429

eBay rate-limits like any public site. `ebay` already paces requests and retries
the transient failures, but a hard limit still means backing off, and it reports
rate-limited (exit 5). Raise the delay between requests with `--rate` (for
example `--rate 1s`) and retry later. A burst of 429 or 5xx responses is the
site asking you to slow down, not a defect.

## A reference is not found

An unknown id, a removed listing, or a reference `ebay` cannot classify reports
not-found (exit 6). Check that the id is spelled the way eBay uses it, and that
the listing still exists in a private browser window before assuming it is gone.

## Prices come back in an unexpected currency

eBay serves prices in the currency tied to the network's location, so the same
page can read in USD, GBP, or VND depending on where you run from. Each record
carries an explicit `currency` field alongside the number, so always read the
two together rather than assuming a currency.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and
a release archive leaves it wherever you unpacked it. If your shell cannot find
`ebay`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).

## Seeing what ebay actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see
the URLs it hit and the responses it got. That is usually enough to tell a bot
wall apart from a rate limit apart from a genuinely empty result. Add
`--no-cache` to force a fresh fetch when you suspect a stale cached page.
