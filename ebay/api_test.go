package ebay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// apiFake stands in for both eBay hosts the Browse backend talks to: the OAuth
// token endpoint and the Browse API. It counts token mints so the test can prove
// the token is cached across calls.
type apiFake struct {
	srv    *httptest.Server
	tokens int
}

func newAPIFake() *apiFake {
	f := &apiFake{}
	mux := http.NewServeMux()
	mux.HandleFunc("/identity/v1/oauth2/token", func(w http.ResponseWriter, _ *http.Request) {
		f.tokens++
		_, _ = w.Write([]byte(`{"access_token":"tok-123","expires_in":7200}`))
	})
	mux.HandleFunc("/buy/browse/v1/item_summary/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"itemSummaries":[
			{"legacyItemId":"123456789012","title":"Apple iPhone 13",
			 "price":{"value":"499.99","currency":"USD"},"condition":"New",
			 "buyingOptions":["FIXED_PRICE"],
			 "itemWebUrl":"https://www.ebay.com/itm/123456789012",
			 "seller":{"username":"techbargains"},
			 "shippingOptions":[{"shippingCost":{"value":"0.00","currency":"USD"}}]}
		]}`))
	})
	mux.HandleFunc("/buy/browse/v1/item/get_item_by_legacy_id", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"legacyItemId":"123456789012","title":"Apple iPhone 13",
			"price":{"value":"549.99","currency":"USD"},"condition":"New",
			"buyingOptions":["FIXED_PRICE"],
			"seller":{"username":"techbargains","feedbackScore":12345,"feedbackPercentage":"99.2"},
			"itemLocation":{"city":"San Jose","country":"US"},
			"categoryPath":"Cell Phones & Accessories|Cell Phones & Smartphones",
			"estimatedAvailabilities":{"estimatedAvailableQuantity":7,"estimatedSoldQuantity":42}}`))
	})
	f.srv = httptest.NewServer(mux)
	return f
}

func (f *apiFake) close() { f.srv.Close() }

// apiClient points an opt-in Browse client at the fake hosts.
func (f *apiFake) apiClient() *apiClient {
	a := newAPIClient(Config{
		ClientID:     "id",
		ClientSecret: "secret",
		Marketplace:  "EBAY_US",
		Timeout:      0,
	}, nil)
	a.tokenURL = f.srv.URL + "/identity/v1/oauth2/token"
	a.browseBase = f.srv.URL + "/buy/browse/v1"
	return a
}

func TestAPISearchItems(t *testing.T) {
	f := newAPIFake()
	defer f.close()
	a := f.apiClient()

	got, err := a.SearchItems(context.Background(), "iphone", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 listing, got %d", len(got))
	}
	l := got[0]
	if l.ID != "123456789012" || l.Title != "Apple iPhone 13" {
		t.Errorf("listing = %+v", l)
	}
	if l.Price != 499.99 || l.Currency != "USD" {
		t.Errorf("price = %v %q", l.Price, l.Currency)
	}
	if l.Buying != "fixed" {
		t.Errorf("buying = %q", l.Buying)
	}
	if l.Seller != "techbargains" {
		t.Errorf("seller = %q", l.Seller)
	}
	if l.Shipping != "Free" {
		t.Errorf("free shipping should map to Free, got %q", l.Shipping)
	}
}

func TestAPIGetItem(t *testing.T) {
	f := newAPIFake()
	defer f.close()
	a := f.apiClient()

	it, err := a.GetItem(context.Background(), "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	if it.Price != 549.99 || it.Condition != "New" {
		t.Errorf("item = %+v", it)
	}
	if it.SellerScore != 12345 || it.SellerRating != 99.2 {
		t.Errorf("seller feedback = %d / %v", it.SellerScore, it.SellerRating)
	}
	if it.Location != "San Jose US" {
		t.Errorf("location = %q", it.Location)
	}
	if it.Category != "Cell Phones & Smartphones" {
		t.Errorf("category leaf = %q", it.Category)
	}
	if it.Available != 7 || it.Sold != 42 {
		t.Errorf("availability = %d / %d", it.Available, it.Sold)
	}
}

func TestAPITokenCached(t *testing.T) {
	f := newAPIFake()
	defer f.close()
	a := f.apiClient()

	for i := 0; i < 3; i++ {
		if _, err := a.SearchItems(context.Background(), "iphone", 5); err != nil {
			t.Fatal(err)
		}
	}
	if f.tokens != 1 {
		t.Errorf("token should be minted once and cached, minted %d times", f.tokens)
	}
}

func TestAPIBadCredentialsAreBlocked(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
	}))
	defer ts.Close()
	a := newAPIClient(Config{ClientID: "x", ClientSecret: "y"}, nil)
	a.tokenURL = ts.URL

	_, err := a.SearchItems(context.Background(), "iphone", 5)
	if err != ErrBlocked {
		t.Errorf("bad credentials should read as ErrBlocked, got %v", err)
	}
}

// TestSearchFallsBackToAPI proves the walled web search routes to the Browse
// backend when credentials are configured: the page host returns the bot wall,
// the API host returns results.
func TestSearchFallsBackToAPI(t *testing.T) {
	f := newAPIFake()
	defer f.close()
	wall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Pardon Our Interruption" + strings.Repeat(" ", 3000)))
	}))
	defer wall.Close()

	c := NewClient(Config{
		BaseURL:      wall.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		Marketplace:  "EBAY_US",
	})
	c.api.tokenURL = f.srv.URL + "/identity/v1/oauth2/token"
	c.api.browseBase = f.srv.URL + "/buy/browse/v1"

	got, err := c.Search(context.Background(), "iphone", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "123456789012" {
		t.Fatalf("fallback results = %+v", got)
	}
}
