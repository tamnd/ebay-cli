package ebay

// This file holds the exported records the commands emit. Their json tags name
// the fields a reader sees, kit:"id" marks the key the record store upserts on,
// and table:",truncate" keeps wide free text from blowing up a terminal table.
// Each record carries only fields a logged-out reader can actually fill: no
// bids you placed, no offers made to you, no seller dashboards, no completed
// sale price history (that surface is itself behind a sign-in wall). There is no
// Rank column either; emit order is the rank, and a stable id is a better store
// key than a position that shifts on every refresh. Several records are emitted
// by more than one surface, and not every surface fills every field; omitempty
// carries the gaps. The per-surface files (search.go, item.go, ...) hold the
// parsing these map from.

// Listing is a summary row in a grid, emitted by search, category browse, and
// seller listings.
type Listing struct {
	ID        string  `json:"id" kit:"id"`
	Title     string  `json:"title,omitempty" table:",truncate"`
	Price     float64 `json:"price,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	Condition string  `json:"condition,omitempty"`
	Buying    string  `json:"buying,omitempty"` // fixed, auction, best_offer
	Bids      int     `json:"bids,omitempty"`   // auctions only
	Shipping  string  `json:"shipping,omitempty"`
	Seller    string  `json:"seller,omitempty"`
	Rating    float64 `json:"rating,omitempty"`  // aggregate rating (category JSON-LD)
	Reviews   int     `json:"reviews,omitempty"` // review count (category JSON-LD)
	Thumbnail string  `json:"thumbnail,omitempty" table:",truncate"`
	URL       string  `json:"url"`
}

// Item is the full detail for one item, emitted by item. The fields are what the
// /itm page shows a logged-out visitor, the same fields the Browse API returns
// on the fallback path.
type Item struct {
	ID           string  `json:"id" kit:"id"`
	Title        string  `json:"title,omitempty" table:",truncate"`
	Subtitle     string  `json:"subtitle,omitempty" table:",truncate"`
	Price        float64 `json:"price,omitempty"`
	Currency     string  `json:"currency,omitempty"`
	Condition    string  `json:"condition,omitempty"`
	Buying       string  `json:"buying,omitempty"` // fixed, auction
	Bids         int     `json:"bids,omitempty"`   // auctions only
	EndsAt       string  `json:"ends_at,omitempty"`
	Available    int     `json:"available,omitempty"`
	Sold         int     `json:"sold,omitempty"`
	Seller       string  `json:"seller,omitempty"`
	SellerScore  int     `json:"seller_score,omitempty"`
	SellerRating float64 `json:"seller_rating,omitempty"`
	Location     string  `json:"location,omitempty"`
	Shipping     string  `json:"shipping,omitempty"`
	Returns      string  `json:"returns,omitempty"`
	Category     string  `json:"category,omitempty"`
	Image        string  `json:"image,omitempty" table:",truncate"`
	URL          string  `json:"url"`
}

// Seller is a seller's public profile, emitted by seller show.
type Seller struct {
	Username  string  `json:"username" kit:"id"`
	Store     string  `json:"store,omitempty"`
	Score     int     `json:"score,omitempty"`    // feedback score (count), when shown
	Positive  float64 `json:"positive,omitempty"` // percent positive feedback
	Sold      int     `json:"sold,omitempty"`     // lifetime items sold (storefront)
	Followers int     `json:"followers,omitempty"`
	Location  string  `json:"location,omitempty"`
	URL       string  `json:"url"`
}

// Category is a category node, emitted by category show and category tree.
type Category struct {
	ID     string `json:"id" kit:"id"` // numeric leaf category id
	Name   string `json:"name"`
	Parent string `json:"parent,omitempty"` // parent category name (from the trail)
	Node   string `json:"node,omitempty"`   // bn_<digits> browse-node id when known
	URL    string `json:"url"`
}

// Deal is a row on the deals hub, emitted by deals.
type Deal struct {
	ID       string  `json:"id" kit:"id"`
	Title    string  `json:"title,omitempty" table:",truncate"`
	Price    float64 `json:"price,omitempty"`
	Was      float64 `json:"was,omitempty"` // original price
	Currency string  `json:"currency,omitempty"`
	Discount string  `json:"discount,omitempty"` // the badge text, e.g. "20% off"
	URL      string  `json:"url"`
}

// Suggestion is one autocomplete term, emitted by suggest.
type Suggestion struct {
	Query string `json:"query"`         // the prefix that was queried
	Term  string `json:"term" kit:"id"` // a suggested completion
}

// Ref is the result of `ebay ref id`: the canonical (kind, id) a reference
// resolves to, plus the live URL, all without touching the network.
type Ref struct {
	Input string `json:"input"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}
