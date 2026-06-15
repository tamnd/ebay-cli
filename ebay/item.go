package ebay

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// item.go reads one item by id. The /itm/ page is one of the two surfaces eBay
// walls from datacenter IPs, so this is best-effort: the client GETs the page
// and, when the bot wall is up (ErrBlocked) and Browse API credentials are set,
// falls back to the documented API for the same record. With no credentials the
// wall propagates as exit 4 with a message that names both remedies.

// GetItem returns one item by id (or /itm/ URL).
func (c *Client) GetItem(ctx context.Context, ref string) (*Item, error) {
	id := itemID(ref)
	if id == "" {
		return nil, ErrNotFound
	}
	body, err := c.get(ctx, c.BaseURL+"/itm/"+id)
	if err != nil || tooSmall(body) {
		if err == nil {
			err = ErrBlocked
		}
		if errors.Is(err, ErrBlocked) && c.api != nil {
			return c.api.GetItem(ctx, id)
		}
		return nil, err
	}
	doc, derr := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if derr != nil {
		return nil, derr
	}
	return parseItem(doc, id), nil
}

// itemID reduces a reference to its bare item id.
func itemID(ref string) string {
	r := Classify(ref)
	if r.Kind == "item" {
		return r.ID
	}
	return strings.Trim(ref, "/")
}

// parseItem lifts the fields the /itm page shows a logged-out visitor.
func parseItem(doc *goquery.Document, id string) *Item {
	it := &Item{ID: id, URL: BaseURL + "/itm/" + id}
	root := doc.Selection

	it.Title = squish(firstText(root, "h1.x-item-title__mainTitle", ".x-item-title__mainTitle", "h1"))
	it.Subtitle = squish(firstText(root, ".x-item-title__subTitle", "[class*=subTitle]"))
	it.Price, it.Currency = parsePrice(firstText(root, ".x-price-primary", "[class*=price-primary]", "[itemprop=price]"))
	it.Condition = squish(firstText(root, ".x-item-condition-text", "[class*=condition]"))

	if t := firstText(root, ".x-bid-count", "[class*=bidCount]"); t != "" {
		it.Bids = parseInt(t)
		it.Buying = "auction"
	} else {
		it.Buying = "fixed"
	}
	it.EndsAt = squish(firstText(root, ".x-end-time", "[class*=endTime]"))

	if t := firstText(root, ".x-quantity__availability", "[class*=availability]"); t != "" {
		it.Available = parseInt(t)
	}
	if t := bodyMatch([]byte(squish(root.Find(".x-quantity, [class*=quantity]").Text())), "sold"); t != "" {
		it.Sold = parseInt(t)
	}

	it.Seller = squish(firstText(root, ".x-sellercard-atf__info__about-seller", "[class*=seller-name]", "[class*=mbg-nw]"))
	if t := firstText(root, ".x-sellercard-atf__data-item", "[class*=feedback]"); t != "" {
		it.SellerScore = parseInt(t)
	}
	if t := bodyMatch([]byte(squish(root.Find(".x-sellercard-atf, [class*=sellercard]").Text())), "% positive"); t != "" {
		it.SellerRating = parseFloat(t)
	}

	it.Location = squish(firstText(root, ".ux-labels-values--shipsFrom .ux-textspans", "[class*=itemLocation]"))
	it.Shipping = squish(firstText(root, ".ux-labels-values--shipping .ux-textspans--BOLD", "[class*=shipping]"))
	it.Returns = squish(firstText(root, ".ux-labels-values--returns .ux-textspans", "[class*=returns]"))
	it.Category = lastCrumb(doc)
	if src := firstAttr(root.Find(".ux-image-carousel img, [class*=image] img").First(), "src", "data-src"); src != "" {
		it.Image = src
	}
	return it
}

// lastCrumb returns the leaf category from the page breadcrumb (JSON-LD or the
// visible trail).
func lastCrumb(doc *goquery.Document) string {
	if trail := breadcrumb(parseJSONLD(doc)); len(trail) > 0 {
		return trail[len(trail)-1]
	}
	var last string
	doc.Find(".seo-breadcrumb-text, [class*=breadcrumb] a").Each(func(_ int, s *goquery.Selection) {
		if t := squish(s.Text()); t != "" {
			last = t
		}
	})
	return last
}
