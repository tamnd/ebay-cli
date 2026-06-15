package ebay

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// cards.go parses the listing grids the category, search, seller, and deals
// pages share. eBay renders each item as an HTML card with an /itm/ link, a
// title, a price, and the condition and format lines. The markup varies a little
// across surfaces (the classic .s-item card, the newer .s-card and
// .brwrvr__item-card), so parseCards tries each container selector and keeps the
// one that yields the most cards.

var cardSelectors = []string{
	"li.s-item",
	".s-card",
	".brwrvr__item-card",
	".str-item-card",
}

// parseCards reduces a listing-grid document to Listing records, capped to limit
// (limit <= 0 means all). It dedupes by item id, since eBay repeats a card in
// hidden carousels.
func parseCards(doc *goquery.Document, limit int) []*Listing {
	var best *goquery.Selection
	bestN := 0
	for _, sel := range cardSelectors {
		s := doc.Find(sel)
		if s.Length() > bestN {
			best, bestN = s, s.Length()
		}
	}
	if best == nil {
		return nil
	}

	var out []*Listing
	seen := map[string]bool{}
	best.EachWithBreak(func(_ int, card *goquery.Selection) bool {
		l := cardToListing(card)
		if l == nil || seen[l.ID] {
			return true
		}
		seen[l.ID] = true
		out = append(out, l)
		return !(limit > 0 && len(out) >= limit)
	})
	return out
}

// cardToListing reads one card. It returns nil when the card carries no item
// link (a promo or a header cell).
func cardToListing(card *goquery.Selection) *Listing {
	href, _ := card.Find(`a[href*="/itm/"]`).First().Attr("href")
	if href == "" {
		// Some cards put the link on the card root.
		href, _ = card.Find("a").First().Attr("href")
	}
	id := itemIDFrom(href)
	if id == "" {
		return nil
	}
	l := &Listing{ID: id, URL: BaseURL + "/itm/" + id}

	l.Title = squish(firstText(card,
		".s-item__title", ".s-card__title", ".bsig__title",
		".str-item-card__property-title", ".str-card-title", "[role=heading]"))
	if l.Title == "" {
		// Storefront cards put the title on the card button's title attribute.
		l.Title = squish(firstAttr(card.Find("[title]").First(), "title"))
	}
	l.Title = strings.TrimPrefix(l.Title, "New Listing")
	l.Title = strings.TrimSpace(l.Title)

	priceTxt := firstText(card,
		".s-item__price", ".s-card__price", ".bsig__price--displayprice", ".bsig__price",
		".str-item-card__property-displayPrice", "[class*=displayPrice]", "[class*=price]")
	l.Price, l.Currency = parsePrice(priceTxt)

	l.Condition = squish(firstText(card,
		".SECONDARY_INFO", ".bsig__listingCondition", ".s-item__subtitle", "[class*=ondition]"))

	if fmtTxt := firstText(card, ".s-item__purchase-options", ".bsig__purchaseOptions", ".s-item__dynamic", "[class*=urchaseOption]"); fmtTxt != "" {
		l.Buying = buyingFormat(fmtTxt)
	}
	if bidTxt := firstText(card, ".s-item__bids", "[class*=bid]"); bidTxt != "" {
		if strings.Contains(strings.ToLower(bidTxt), "bid") {
			l.Bids = parseInt(bidTxt)
			if l.Buying == "" {
				l.Buying = "auction"
			}
		}
	}
	l.Shipping = squish(firstText(card,
		".s-item__shipping", ".s-item__logisticsCost", ".bsig__logisticsCost", "[class*=ogisticsCost]", "[class*=hipping]"))
	l.Seller = squish(firstText(card, ".s-item__seller-info-text", ".bsig__sellerInfo", "[class*=ellerInfo]"))

	if src := firstAttr(card.Find("img").First(), "src", "data-src", "data-defer-load"); src != "" {
		l.Thumbnail = src
	}
	return l
}

// firstText returns the trimmed text of the first selector that matches.
func firstText(s *goquery.Selection, selectors ...string) string {
	for _, sel := range selectors {
		if m := s.Find(sel).First(); m.Length() > 0 {
			if t := strings.TrimSpace(m.Text()); t != "" {
				return t
			}
		}
	}
	return ""
}

// firstAttr returns the first non-empty attribute among names on a selection.
func firstAttr(s *goquery.Selection, names ...string) string {
	for _, n := range names {
		if v, ok := s.Attr(n); ok && v != "" {
			return v
		}
	}
	return ""
}
