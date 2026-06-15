package ebay_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/ebay-cli/ebay"
)

// newClient points a client at a fake server with no pacing or retries.
func newClient(ts *httptest.Server) *ebay.Client {
	cfg := ebay.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Delay = 0
	cfg.Retries = 0
	return ebay.NewClient(cfg)
}

// serve returns a server that replies with body for every request.
func serve(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
}

// padded pads a body past the 2KB too-small threshold so the wall heuristics do
// not misfire on a small fixture.
func padded(s string) string {
	return s + "<!-- " + strings.Repeat("x", 2200) + " -->"
}

// categoryPage mirrors the real /b/ markup: a multi-level BreadcrumbList and a
// CollectionPage whose about.offers.itemOffered carries one Product per grid item
// (price, image gallery, and a rating), with the listing cards in the body.
const categoryPage = `<html><head>
<script type="application/ld+json">
{"@type":"BreadcrumbList","itemListElement":[
  {"position":1,"name":"eBay"},
  {"position":2,"name":"Cell Phones & Accessories"},
  {"position":3,"name":"Cell Phones & Smartphones"}]}
</script>
<script type="application/ld+json">
{"@type":"CollectionPage","about":{"@type":"WebPage","offers":{"@type":"Offer","itemOffered":[
  {"@type":"Product","url":"https://www.ebay.com/itm/111111111111",
   "image":["https://i.ebayimg.com/a.jpg","https://i.ebayimg.com/b.jpg"],
   "offers":{"@type":"Offer","price":499.99,"priceCurrency":"USD"},
   "aggregateRating":{"ratingValue":"4.8","reviewCount":"230"}}
]}}}
</script>
</head><body>
<ul>
<li class="s-item">
  <a href="https://www.ebay.com/itm/111111111111">x</a>
  <div class="s-item__title">Apple iPhone 13 128GB</div>
  <span class="s-item__price">$499.99</span>
  <span class="SECONDARY_INFO">Pre-owned</span>
  <span class="s-item__shipping">Free shipping</span>
</li>
<li class="s-item">
  <a href="https://www.ebay.com/itm/222222222222">x</a>
  <div class="s-item__title">Samsung Galaxy S22</div>
  <span class="s-item__price">$399.00</span>
  <span class="SECONDARY_INFO">Brand New</span>
  <span class="s-item__hotness">97 sold</span>
  <span class="s-item__watchCount">5 watching</span>
</li>
</ul></body></html>`

func TestCategoryBrowse(t *testing.T) {
	ts := serve(padded(categoryPage))
	defer ts.Close()

	got, err := newClient(ts).CategoryBrowse(context.Background(), "9355", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 listings, got %d", len(got))
	}
	a := got[0]
	if a.ID != "111111111111" || a.Title != "Apple iPhone 13 128GB" {
		t.Errorf("listing 0 = %+v", a)
	}
	if a.Price != 499.99 || a.Currency != "USD" {
		t.Errorf("price = %v %q", a.Price, a.Currency)
	}
	if a.Condition != "Pre-owned" || a.Shipping != "Free shipping" {
		t.Errorf("condition/shipping = %q / %q", a.Condition, a.Shipping)
	}
	if a.Rating != 4.8 || a.Reviews != 230 {
		t.Errorf("rating from JSON-LD = %v (%d reviews)", a.Rating, a.Reviews)
	}
	if len(a.Images) != 2 || a.Images[0] != "https://i.ebayimg.com/a.jpg" {
		t.Errorf("image gallery from JSON-LD = %v", a.Images)
	}
	if a.URL != "https://www.ebay.com/itm/111111111111" {
		t.Errorf("url = %q", a.URL)
	}
	b := got[1]
	if b.Sold != 97 || b.Watching != 5 {
		t.Errorf("sold/watching from card = %d / %d", b.Sold, b.Watching)
	}
}

func TestGetCategory(t *testing.T) {
	ts := serve(padded(categoryPage))
	defer ts.Close()

	cat, err := newClient(ts).GetCategory(context.Background(), "9355")
	if err != nil {
		t.Fatal(err)
	}
	if cat.Name != "Cell Phones & Smartphones" {
		t.Errorf("name = %q", cat.Name)
	}
	if cat.Parent != "Cell Phones & Accessories" {
		t.Errorf("parent = %q", cat.Parent)
	}
	if cat.ID != "9355" || cat.URL != "https://www.ebay.com/b/9355" {
		t.Errorf("id/url = %q / %q", cat.ID, cat.URL)
	}
	wantTrail := []string{"eBay", "Cell Phones & Accessories", "Cell Phones & Smartphones"}
	if len(cat.Trail) != len(wantTrail) {
		t.Fatalf("trail = %v", cat.Trail)
	}
	for i, name := range wantTrail {
		if cat.Trail[i] != name {
			t.Errorf("trail[%d] = %q, want %q", i, cat.Trail[i], name)
		}
	}
}

// sellerPage mirrors a real storefront header, where each figure sits in its
// own span and the follower and sold counts are K/M-abbreviated.
const sellerPage = `<html><head>
<meta property="og:image" content="https://i.ebayimg.com/store-logo.jpg" />
</head><body>
<div class="str-seller-card-wrap">
  <h1 class="str-seller-card__store-name">Tech Bargains Store</h1>
  <span>99.4%</span> <span>positive feedback</span>
  <span>768K</span> <span>items sold</span>
  <span>78K</span> <span>followers</span>
  <div class="str-seller-card__location">San Jose, CA</div>
</div>
<script>{"topRatedSeller":{"icon":{"name":"TOP_RATED_SELLER"}}}</script>
<ul>
<li class="s-item"><a href="/itm/333333333333">x</a>
  <div class="s-item__title">USB-C Cable</div><span class="s-item__price">$9.99</span></li>
</ul></body></html>`

func TestGetSeller(t *testing.T) {
	ts := serve(padded(sellerPage))
	defer ts.Close()

	s, err := newClient(ts).GetSeller(context.Background(), "techbargains")
	if err != nil {
		t.Fatal(err)
	}
	if s.Username != "techbargains" {
		t.Errorf("username = %q", s.Username)
	}
	if s.Store != "Tech Bargains Store" {
		t.Errorf("store = %q", s.Store)
	}
	if s.Positive != 99.4 {
		t.Errorf("positive = %v", s.Positive)
	}
	if s.Sold != 768000 {
		t.Errorf("sold (768K expanded) = %d", s.Sold)
	}
	if s.Followers != 78000 {
		t.Errorf("followers (78K expanded) = %d", s.Followers)
	}
	if s.Logo != "https://i.ebayimg.com/store-logo.jpg" {
		t.Errorf("logo from meta = %q", s.Logo)
	}
	if !s.TopRated {
		t.Errorf("top-rated badge not detected")
	}
}

func TestSellerListings(t *testing.T) {
	ts := serve(padded(sellerPage))
	defer ts.Close()

	got, err := newClient(ts).SellerListings(context.Background(), "techbargains", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "333333333333" {
		t.Fatalf("listings = %+v", got)
	}
	if got[0].Seller != "techbargains" {
		t.Errorf("seller not filled from the username: %q", got[0].Seller)
	}
}

// dealsPage mirrors a real deals tile: two /itm/ links (image and title), the
// price under itemprop, and the previous price plus discount in the original
// price block's screen-reader text.
const dealsPage = `<html><body>
<div class="dne-itemtile dne-itemtile-large">
  <a href="/itm/444444444444" class="dne-itemtile-imagewrapper"><img src="x.webp"></a>
  <a href="/itm/444444444444"><h3 class="dne-itemtile-title"><span itemprop="name">Robot Vacuum Cleaner</span></h3></a>
  <div class="dne-itemtile-price"><span itemprop="price">$129.99</span></div>
  <div class="dne-itemtile-original-price">
    <span class="itemtile-price-strikethrough">$199.99</span>
    <span class="clipped">Previous price: $199.99 35% off</span>
  </div>
  <span class="dne-itemcard-hotness">Almost gone</span>
  <span class="dne-itemtile-delivery">Free shipping</span>
</div></body></html>`

func TestDeals(t *testing.T) {
	ts := serve(padded(dealsPage))
	defer ts.Close()

	got, err := newClient(ts).Deals(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 deal, got %d", len(got))
	}
	d := got[0]
	if d.ID != "444444444444" || d.Price != 129.99 || d.Was != 199.99 {
		t.Errorf("deal = %+v", d)
	}
	if d.Discount != "35% off" {
		t.Errorf("discount = %q", d.Discount)
	}
	if d.Image != "x.webp" {
		t.Errorf("image = %q", d.Image)
	}
	if !d.FreeShipping {
		t.Errorf("free shipping not detected")
	}
	if !d.Trending {
		t.Errorf("trending badge not detected")
	}
}

func TestSuggest(t *testing.T) {
	ts := serve(`["iphone", ["iphone 13", "iphone 14", "iphone case"]]`)
	defer ts.Close()

	c := newClient(ts)
	c.SuggestURL = ts.URL // the autocomplete host, redirected to the fake

	got, err := c.Suggest(context.Background(), "iphone", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 suggestions, got %d", len(got))
	}
	if got[0].Term != "iphone 13" || got[0].Query != "iphone" {
		t.Errorf("suggestion 0 = %+v", got[0])
	}
	if got[2].Term != "iphone case" {
		t.Errorf("suggestion 2 = %+v", got[2])
	}
}

func TestSearchWallNoFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Pardon Our Interruption"))
	}))
	defer ts.Close()

	_, err := newClient(ts).Search(context.Background(), "iphone", 10)
	if !errors.Is(err, ebay.ErrBlocked) {
		t.Errorf("walled search without credentials should be ErrBlocked, got %v", err)
	}
}

func TestItemInterstitialIsBlocked(t *testing.T) {
	ts := serve("Pardon Our Interruption" + strings.Repeat(" ", 3000))
	defer ts.Close()

	_, err := newClient(ts).GetItem(context.Background(), "123456789012")
	if !errors.Is(err, ebay.ErrBlocked) {
		t.Errorf("interstitial body should be ErrBlocked, got %v", err)
	}
}

func TestItemNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := newClient(ts).GetItem(context.Background(), "123456789012")
	if !errors.Is(err, ebay.ErrNotFound) {
		t.Errorf("404 should be ErrNotFound, got %v", err)
	}
}

func TestRateLimited(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	_, err := newClient(ts).CategoryBrowse(context.Background(), "9355", 10)
	if !errors.Is(err, ebay.ErrRateLimited) {
		t.Errorf("429 should be ErrRateLimited, got %v", err)
	}
}

func TestItemParse(t *testing.T) {
	const itemPage = `<html><body>
	<h1 class="x-item-title__mainTitle"><span>Apple iPhone 13 128GB Blue Unlocked</span></h1>
	<div class="x-price-primary"><span>$549.99</span></div>
	<div class="x-item-condition-text"><span>Open box</span></div>
	<div class="ux-labels-values--shipping"><span class="ux-textspans--BOLD">Free</span></div>
	<div class="ux-image-carousel">
	  <img src="https://i.ebayimg.com/one.jpg">
	  <img src="https://i.ebayimg.com/two.jpg">
	  <img src="https://i.ebayimg.com/one.jpg">
	</div>
	</body></html>`
	ts := serve(padded(itemPage))
	defer ts.Close()

	it, err := newClient(ts).GetItem(context.Background(), "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != "123456789012" {
		t.Errorf("id = %q", it.ID)
	}
	if it.Title != "Apple iPhone 13 128GB Blue Unlocked" {
		t.Errorf("title = %q", it.Title)
	}
	if it.Price != 549.99 || it.Currency != "USD" {
		t.Errorf("price = %v %q", it.Price, it.Currency)
	}
	if it.Condition != "Open box" {
		t.Errorf("condition = %q", it.Condition)
	}
	// The carousel lists two distinct images (one repeated); the gallery dedupes
	// in order and Image keeps the first.
	if len(it.Images) != 2 || it.Images[0] != "https://i.ebayimg.com/one.jpg" {
		t.Errorf("image gallery = %v", it.Images)
	}
	if it.Image != "https://i.ebayimg.com/one.jpg" {
		t.Errorf("image = %q", it.Image)
	}
}
