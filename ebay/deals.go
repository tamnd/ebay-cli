package ebay

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// offRE pulls the "30% off" badge out of a deal tile's price block, where the
// discount is printed in the screen-reader text alongside the previous price.
var offRE = regexp.MustCompile(`[0-9]+% off`)

// deals.go reads the deals hub (/deals), a reliable-core surface eBay serves
// anonymously. The cards carry a title, the deal price, the original price, and a
// discount badge; the parser reuses the shared card reader for the common fields
// and lifts the was-price and badge per card.

// Deals returns the current deals.
func (c *Client) Deals(ctx context.Context, limit int) ([]*Deal, error) {
	body, err := c.get(ctx, c.BaseURL+"/deals")
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var out []*Deal
	seen := map[string]bool{}
	doc.Find(`a[href*="/itm/"]`).EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")
		id := itemIDFrom(href)
		if id == "" || seen[id] {
			return true
		}
		// Climb to the tile so the price and discount come from this item. Each
		// tile carries two /itm/ links (image and title); the dedupe by id keeps
		// one, and climbing to the tile (not the image wrapper, which also has
		// "item" in its class) reaches the title and price either way.
		card := a.Closest(".dne-itemtile, .dne-itemcard, li")
		if card.Length() == 0 {
			card = a.Parent()
		}
		d := &Deal{ID: id, URL: BaseURL + "/itm/" + id}
		d.Title = squish(firstText(card, ".dne-itemtile-title", "[itemprop=name]", "h3"))
		if d.Title == "" {
			d.Title = squish(a.Text())
		}
		d.Price, d.Currency = parsePrice(firstText(card,
			".dne-itemtile-price [itemprop=price]", "[itemprop=price]", ".dne-itemtile-price"))
		orig := card.Find(".dne-itemtile-original-price")
		if was := squish(orig.Find(".itemtile-price-strikethrough").Text()); was != "" {
			d.Was, _ = parsePrice(was)
		}
		if m := offRE.FindString(orig.Text()); m != "" {
			d.Discount = m
		}
		if src := firstAttr(card.Find("img").First(), "src", "data-src", "data-defer-load"); src != "" {
			d.Image = src
		}
		if strings.Contains(strings.ToLower(card.Text()), "free shipping") {
			d.FreeShipping = true
		}
		if card.Find(".dne-itemcard-hotness, [class*=hotness]").Length() > 0 {
			d.Trending = true
		}
		seen[id] = true
		out = append(out, d)
		return limit <= 0 || len(out) < limit
	})
	return out, nil
}
