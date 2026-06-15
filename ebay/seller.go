package ebay

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// The storefront figures, each anchored on its own label so they never pick up
// a neighbour's number. positiveRE reads the percent, soldRE the lifetime sold
// count, followersRE the follower count (both K/M-abbreviated), and scoreRE the
// parenthesized feedback-score count on the pages that still print it.
var (
	positiveRE  = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*%\s*positive`)
	soldRE      = regexp.MustCompile(`([0-9][0-9,]*\.?[0-9]*\s*[KMB]?)\s*items?\s*sold`)
	followersRE = regexp.MustCompile(`([0-9][0-9,]*\.?[0-9]*\s*[KMB]?)\s*followers`)
	scoreRE     = regexp.MustCompile(`[Ff]eedback\s*score(?:\s*of)?\s*([0-9][0-9,]*)`)
)

// seller.go reads a seller's public store page, a reliable-core surface eBay
// serves anonymously. /usr/<name> redirects to the store page where one exists;
// the client follows redirects, so either form works. The page yields the store
// name, the feedback score and percent, the follower count, and the item cards.

// sellerURL builds the seller store URL from a username, store slug, or URL.
func (c *Client) sellerURL(ref string) string {
	r := Classify(ref)
	if r.Kind == "seller" {
		return c.BaseURL + "/usr/" + r.ID
	}
	return c.BaseURL + "/usr/" + strings.Trim(ref, "/")
}

func sellerName(ref string) string {
	r := Classify(ref)
	if r.Kind == "seller" {
		return r.ID
	}
	return strings.Trim(ref, "/")
}

// GetSeller returns a seller's profile.
func (c *Client) GetSeller(ctx context.Context, ref string) (*Seller, error) {
	name := sellerName(ref)
	body, err := c.get(ctx, c.sellerURL(ref))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	s := &Seller{Username: name, URL: BaseURL + "/usr/" + name}

	s.Store = squish(firstText(doc.Selection,
		".str-seller-card__store-name", ".str-billboard__title", "h1.shop-header__title", "h1"))

	// The storefront header prints "<name> 99.4% positive feedback 768K items
	// sold 78K followers", each figure in its own span. Reading the figures off
	// the card's collapsed text, rather than the raw bytes, joins the spans so a
	// percent and its label sit next to each other. eBay no longer shows the raw
	// feedback-score count there, so Score stays empty unless the page still
	// prints one; the K/M abbreviations are expanded.
	hdr := squish(firstText(doc.Selection,
		".str-seller-card-wrap", ".str-seller-card", "[class*=seller-card]"))
	if hdr == "" {
		hdr = squish(doc.Selection.Find("body").Text())
	}
	if m := positiveRE.FindStringSubmatch(hdr); m != nil {
		s.Positive = parseFloat(m[1])
	}
	if m := soldRE.FindStringSubmatch(hdr); m != nil {
		s.Sold = parseHuman(m[1])
	}
	if m := followersRE.FindStringSubmatch(hdr); m != nil {
		s.Followers = parseHuman(m[1])
	}
	if m := scoreRE.FindStringSubmatch(hdr); m != nil {
		s.Score = parseInt(m[1])
	}
	s.Location = squish(firstText(doc.Selection, ".str-seller-card__location", "[class*=location]"))
	return s, nil
}

// SellerListings returns a seller's active listings.
func (c *Client) SellerListings(ctx context.Context, ref string, limit int) ([]*Listing, error) {
	body, err := c.get(ctx, c.sellerURL(ref))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	listings := parseCards(doc, limit)
	seller := sellerName(ref)
	for _, l := range listings {
		if l.Seller == "" {
			l.Seller = seller
		}
	}
	return listings, nil
}

// bodyMatch returns a short window of text around the first occurrence of marker,
// so a number printed next to a label ("12,345 followers") can be parsed out.
func bodyMatch(body []byte, marker string) string {
	i := bytes.Index(body, []byte(marker))
	if i < 0 {
		return ""
	}
	start := i - 24
	if start < 0 {
		start = 0
	}
	end := i + len(marker)
	return stripTags(body[start:end])
}

// stripTags reduces an HTML fragment to its text, collapsing whitespace.
func stripTags(b []byte) string {
	var out strings.Builder
	depth := 0
	for _, c := range b {
		switch c {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				out.WriteByte(c)
			}
		}
	}
	return squish(out.String())
}
