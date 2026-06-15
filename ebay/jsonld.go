package ebay

import (
	"encoding/json"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// jsonld.go reads the schema.org JSON-LD eBay embeds in its pages. The category
// browse page carries a BreadcrumbList for the category trail and a CollectionPage
// whose about.offers.itemOffered holds one Product per grid item, each with a
// price, an image gallery, and sometimes an aggregate rating. The parsers here
// lift the trail and a per-item data map, which category.go folds onto the cards.

// ldOffer is the schema.org Offer. On a Product it carries the price; on the
// CollectionPage's about.offers it instead wraps the per-item Product list.
type ldOffer struct {
	Price         json.RawMessage `json:"price"`
	PriceCurrency string          `json:"priceCurrency"`
	ItemOffered   []ldNode        `json:"itemOffered"`
}

// ldNode is the loose shape we read from a JSON-LD block. eBay mixes a single
// object, an array of objects, and an @graph wrapper, and nests the grid Products
// under a CollectionPage rather than at the top level, so the fields are all
// optional and the walkers handle each form.
type ldNode struct {
	Type            json.RawMessage `json:"@type"`
	Graph           []ldNode        `json:"@graph"`
	URL             string          `json:"url"`
	ItemListElement []ldNode        `json:"itemListElement"`
	Position        int             `json:"position"`
	Name            string          `json:"name"`
	Item            *ldNode         `json:"item"`
	Image           json.RawMessage `json:"image"`  // a string or an array of strings
	About           *ldNode         `json:"about"`  // CollectionPage.about (a WebPage)
	Offers          *ldOffer        `json:"offers"` // a Product's price, or the page's itemOffered list
	AggregateRating *struct {
		RatingValue json.RawMessage `json:"ratingValue"`
		ReviewCount json.RawMessage `json:"reviewCount"`
		RatingCount json.RawMessage `json:"ratingCount"`
	} `json:"aggregateRating"`
}

// parseJSONLD returns every top-level JSON-LD node on the page, flattening any
// @graph wrappers.
func parseJSONLD(doc *goquery.Document) []ldNode {
	var nodes []ldNode
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		raw := []byte(s.Text())
		// A block is either an object or an array of objects.
		var one ldNode
		if err := json.Unmarshal(raw, &one); err == nil && (one.Type != nil || len(one.Graph) > 0 || one.URL != "") {
			nodes = append(nodes, flatten(one)...)
			return
		}
		var many []ldNode
		if err := json.Unmarshal(raw, &many); err == nil {
			for _, n := range many {
				nodes = append(nodes, flatten(n)...)
			}
		}
	})
	return nodes
}

func flatten(n ldNode) []ldNode {
	if len(n.Graph) > 0 {
		out := make([]ldNode, 0, len(n.Graph))
		for _, g := range n.Graph {
			out = append(out, flatten(g)...)
		}
		return out
	}
	return []ldNode{n}
}

// productInfo is the per-item data a category page's JSON-LD carries beyond the
// HTML card: a canonical price and currency, the image gallery, and an aggregate
// rating when one is present.
type productInfo struct {
	price    float64
	currency string
	images   []string
	rating   float64
	reviews  int
}

// productData builds the item-id to productInfo map from the Product nodes on a
// page, reaching the ones nested under the CollectionPage as well as any at the
// top level. It merges fields, so a Product seen twice fills whatever was empty.
func productData(nodes []ldNode) map[string]productInfo {
	m := map[string]productInfo{}
	for _, n := range collectProducts(nodes) {
		id := itemIDFrom(n.URL)
		if id == "" {
			continue
		}
		info := m[id]
		if n.Offers != nil {
			if v := ldFloat(n.Offers.Price); v != 0 {
				info.price = v
			}
			if n.Offers.PriceCurrency != "" {
				info.currency = n.Offers.PriceCurrency
			}
		}
		if imgs := ldImages(n.Image); len(imgs) > 0 {
			info.images = imgs
		}
		if n.AggregateRating != nil {
			info.rating = ldFloat(n.AggregateRating.RatingValue)
			info.reviews = ldInt(firstRaw(n.AggregateRating.ReviewCount, n.AggregateRating.RatingCount))
		}
		m[id] = info
	}
	return m
}

// collectProducts gathers every Product node reachable from the page: the ones
// at the top level and, since eBay nests the grid under a CollectionPage, the
// ones under about.offers.itemOffered and inside any itemListElement.
func collectProducts(nodes []ldNode) []ldNode {
	var out []ldNode
	var walk func(n ldNode)
	walk = func(n ldNode) {
		if n.URL != "" && (ldHasType(n, "Product") || n.AggregateRating != nil ||
			(n.Offers != nil && len(n.Offers.Price) > 0)) {
			out = append(out, n)
		}
		if n.About != nil {
			walk(*n.About)
		}
		if n.Offers != nil {
			for _, p := range n.Offers.ItemOffered {
				walk(p)
			}
		}
		for _, e := range n.ItemListElement {
			if e.Item != nil {
				walk(*e.Item)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return out
}

// ldImages reads a schema.org image value, which eBay encodes as a single URL
// string on some pages and an array of URLs on others.
func ldImages(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var one string
	if json.Unmarshal(raw, &one) == nil && one != "" {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return many
	}
	return nil
}

// metaContent returns the content of the first <meta> tag matching any key,
// trying both the property= and name= forms eBay uses across its pages.
func metaContent(doc *goquery.Document, keys ...string) string {
	for _, k := range keys {
		for _, attr := range []string{"property", "name"} {
			if v, ok := doc.Find(`meta[` + attr + `="` + k + `"]`).First().Attr("content"); ok && v != "" {
				return v
			}
		}
	}
	return ""
}

// breadcrumb returns the category trail names in order, from the BreadcrumbList
// node, or nil when there is none.
func breadcrumb(nodes []ldNode) []string {
	for _, n := range nodes {
		if !ldHasType(n, "BreadcrumbList") || len(n.ItemListElement) == 0 {
			continue
		}
		names := make([]string, 0, len(n.ItemListElement))
		for _, e := range n.ItemListElement {
			name := e.Name
			if name == "" && e.Item != nil {
				name = e.Item.Name
			}
			if name != "" {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

func ldHasType(n ldNode, want string) bool {
	if n.Type == nil {
		return false
	}
	var one string
	if json.Unmarshal(n.Type, &one) == nil {
		return one == want
	}
	var many []string
	if json.Unmarshal(n.Type, &many) == nil {
		for _, t := range many {
			if t == want {
				return true
			}
		}
	}
	return false
}

// ldFloat reads a JSON-LD number that may be encoded as a number or a string.
func ldFloat(raw json.RawMessage) float64 {
	if len(raw) == 0 {
		return 0
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return f
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}
	return 0
}

func ldInt(raw json.RawMessage) int {
	return int(ldFloat(raw))
}

func firstRaw(a, b json.RawMessage) json.RawMessage {
	if len(a) > 0 {
		return a
	}
	return b
}
