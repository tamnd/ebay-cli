package ebay

import "errors"

// The library reports its outcomes as a few sentinel errors. domain.go's mapErr
// translates each into the kit error kind that carries the matching exit code,
// so the standalone binary and a host agree on what a wall, a throttle, and a
// miss mean.
var (
	// ErrNotFound is a missing entity: an unknown item id, an unknown seller, or
	// a bad category. eBay serves these as a 404 or a not-found shell page.
	// Exit code 6.
	ErrNotFound = errors.New("not found")

	// ErrRateLimited is a sustained HTTP 429 after the client's own retries.
	// Slow down with --rate. Exit code 5.
	ErrRateLimited = errors.New("rate limited")

	// ErrBlocked is the Akamai bot wall: a 403 interstitial, a reset, or an
	// implausibly small body on a page that should be large. eBay walls the item
	// page and keyword search from datacenter IPs, so this is expected there and
	// the message names the two remedies (a residential IP, or Browse API
	// credentials). Exit code 4.
	ErrBlocked = errors.New("blocked by eBay's bot wall (retry from a residential network, " +
		"or set EBAY_CLIENT_ID and EBAY_CLIENT_SECRET to use the Browse API)")
)
