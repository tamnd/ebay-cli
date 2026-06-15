package ebay

import (
	"bytes"
	"context"
	"errors"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

// search.go reads keyword search. The /sch/ page is the second surface eBay walls
// from datacenter IPs, so this is best-effort with the same fallback as item.go:
// the client GETs the search page and, when the bot wall is up and Browse API
// credentials are set, falls back to the documented API for the same records.

// Search returns listings matching a keyword query.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]*Listing, error) {
	u := c.BaseURL + "/sch/i.html?_nkw=" + url.QueryEscape(query)
	body, err := c.get(ctx, u)
	if err != nil {
		if errors.Is(err, ErrBlocked) && c.api != nil {
			return c.api.SearchItems(ctx, query, limit)
		}
		return nil, err
	}
	doc, derr := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if derr != nil {
		return nil, derr
	}
	listings := parseCards(doc, limit)
	// An empty parse from a too-small body is the wall serving a stub; treat it
	// as blocked so the fallback (or the honest exit 4) takes over.
	if len(listings) == 0 && tooSmall(body) {
		if c.api != nil {
			return c.api.SearchItems(ctx, query, limit)
		}
		return nil, ErrBlocked
	}
	return listings, nil
}
