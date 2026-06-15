package ebay

import (
	"encoding/json"
	"strconv"

	"github.com/PuerkitoBio/goquery"
)

// jsonld.go reads the schema.org JSON-LD eBay embeds in its pages. The category
// browse page carries a BreadcrumbList for the category trail and Product objects
// with an AggregateRating; the parsers here lift the trail and a rating-by-item
// map, which category.go folds onto the HTML cards.

// ratings maps an item id to its aggregate rating and review count, pulled from
// the Product JSON-LD on a listing page.
type ratings map[string]struct {
	value   float64
	reviews int
}

// ldNode is the loose shape we read from a JSON-LD block. eBay mixes a single
// object, an array of objects, and an @graph wrapper, so the fields are all
// optional and the walkers handle each form.
type ldNode struct {
	Type            json.RawMessage `json:"@type"`
	Graph           []ldNode        `json:"@graph"`
	URL             string          `json:"url"`
	ItemListElement []ldNode        `json:"itemListElement"`
	Position        int             `json:"position"`
	Name            string          `json:"name"`
	Item            *ldNode         `json:"item"`
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

// productRatings builds the rating-by-item map from the Product nodes on a page.
func productRatings(nodes []ldNode) ratings {
	r := ratings{}
	for _, n := range nodes {
		if n.AggregateRating == nil || n.URL == "" {
			continue
		}
		id := itemIDFrom(n.URL)
		if id == "" {
			continue
		}
		r[id] = struct {
			value   float64
			reviews int
		}{
			value:   ldFloat(n.AggregateRating.RatingValue),
			reviews: ldInt(firstRaw(n.AggregateRating.ReviewCount, n.AggregateRating.RatingCount)),
		}
	}
	return r
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
