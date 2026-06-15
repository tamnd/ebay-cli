package ebay

import (
	"regexp"
	"strconv"
	"strings"
)

// html.go holds the shared parsing helpers the page surfaces lean on: pulling an
// item id out of an /itm/ link, turning a displayed price into a number and a
// currency, and reducing the small numbers eBay prints ("1,234 sold",
// "99.2% positive") into typed values.

var (
	itmIDRE = regexp.MustCompile(`/itm/(?:[^/]+/)?(\d{9,13})`)
	priceRE = regexp.MustCompile(`([\$£€])\s*([0-9][0-9,]*\.?[0-9]*)`)
	// isoPriceRE matches a localized price printed as a bare number followed by
	// a three-letter currency code, e.g. "15,103,947.37 VND" or "19.99 USD",
	// the shape eBay serves outside the dollar marketplaces.
	isoPriceRE = regexp.MustCompile(`([0-9][0-9,]*\.?[0-9]*)\s*([A-Z]{3})\b`)
	intRE      = regexp.MustCompile(`[0-9][0-9,]*`)
	floatRE    = regexp.MustCompile(`[0-9]+\.?[0-9]*`)
	currNames  = map[string]string{"$": "USD", "£": "GBP", "€": "EUR"}
)

// itemIDFrom extracts the 9-to-13-digit item id from an /itm/ URL or href, or ""
// when there is none.
func itemIDFrom(href string) string {
	m := itmIDRE.FindStringSubmatch(href)
	if m == nil {
		return ""
	}
	return m[1]
}

// parsePrice turns a displayed price ("$1,299.00", "US $19.99", "£5") into a
// number and an ISO currency. It returns (0, "") when there is no price, so a
// caller can leave the fields empty.
func parsePrice(s string) (float64, string) {
	// A symbol-prefixed price ("$1,299.00", "US $19.99", "£5"). A leading "US "
	// or "C " is ignored; the first symbol wins, so a range takes its low end.
	if m := priceRE.FindStringSubmatch(s); m != nil {
		if v, err := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", ""), 64); err == nil {
			return v, currNames[m[1]]
		}
	}
	// A localized price: a number then an ISO currency code ("15,103,947.37 VND").
	if m := isoPriceRE.FindStringSubmatch(s); m != nil {
		if v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
			return v, m[2]
		}
	}
	return 0, ""
}

// parseInt pulls the first integer out of a string ("1,234 sold" -> 1234), or 0.
func parseInt(s string) int {
	m := intRE.FindString(s)
	if m == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.ReplaceAll(m, ",", ""))
	if err != nil {
		return 0
	}
	return n
}

// parseFloat pulls the first decimal out of a string ("99.2% positive" -> 99.2),
// or 0.
func parseFloat(s string) float64 {
	m := floatRE.FindString(s)
	if m == "" {
		return 0
	}
	f, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0
	}
	return f
}

// humanRE matches an abbreviated count like "78K", "1.2M", or "768K", the form
// eBay prints follower and items-sold figures in.
var humanRE = regexp.MustCompile(`([0-9][0-9,]*\.?[0-9]*)\s*([KMB]?)`)

// parseHuman reads an abbreviated count ("78K" -> 78000, "1.2M" -> 1200000), or
// a plain number, into an int. It returns 0 when there is no number.
func parseHuman(s string) int {
	m := humanRE.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
	if err != nil {
		return 0
	}
	switch m[2] {
	case "K":
		v *= 1e3
	case "M":
		v *= 1e6
	case "B":
		v *= 1e9
	}
	return int(v)
}

// squish collapses runs of whitespace into single spaces and trims the ends, so
// a value lifted from indented HTML reads cleanly.
func squish(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// buyingFormat classifies a card's format line into the record's buying field.
func buyingFormat(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.Contains(l, "bid"):
		return "auction"
	case strings.Contains(l, "best offer"):
		return "best_offer"
	case strings.Contains(l, "buy it now") || strings.Contains(l, "now"):
		return "fixed"
	default:
		return ""
	}
}
