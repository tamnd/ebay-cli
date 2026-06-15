package ebay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// api.go is the opt-in Browse API backend. eBay publishes a REST Browse API that
// returns item and search data as clean JSON. It is not anonymous in the keyless
// sense: it needs a free developer application's Client ID and Client Secret,
// exchanged for an application token via the OAuth2 client_credentials grant
// (no user logs in; the token represents the application, the way the web
// client's public id represents the web app). This CLI never requires it, but
// the item page and keyword search are exactly the surfaces eBay walls from
// datacenter networks, so the walled commands fall back here when the operator
// has set EBAY_CLIENT_ID and EBAY_CLIENT_SECRET.

type apiClient struct {
	HTTP         *http.Client
	ClientID     string
	ClientSecret string
	Marketplace  string

	// tokenURL and browseBase are the OAuth and Browse endpoints. They default
	// to the public eBay hosts; tests point them at an httptest server.
	tokenURL   string
	browseBase string

	cache *cache

	mu  sync.Mutex
	tok string
	exp time.Time
}

func newAPIClient(cfg Config, c *cache) *apiClient {
	market := cfg.Marketplace
	if market == "" {
		market = defaultMarket
	}
	return &apiClient{
		HTTP:         &http.Client{Timeout: cfg.Timeout},
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Marketplace:  market,
		tokenURL:     tokenEndpoint,
		browseBase:   browseBase,
		cache:        c,
	}
}

// token returns a cached application token or mints one via the
// client_credentials grant.
func (a *apiClient) token(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tok != "" && time.Now().Before(a.exp) {
		return a.tok, nil
	}

	form := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {apiScope},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(a.ClientID + ":" + a.ClientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// Bad credentials read as the same wall: the operator's opt-in did not
		// take, so the surface stays blocked.
		return "", ErrBlocked
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &out); err != nil || out.AccessToken == "" {
		return "", ErrBlocked
	}
	a.tok = out.AccessToken
	// Expire a minute early so a request never races the boundary.
	a.exp = time.Now().Add(time.Duration(out.ExpiresIn-60) * time.Second)
	return a.tok, nil
}

// getJSON runs one authenticated GET against the Browse API and decodes it.
func (a *apiClient) getJSON(ctx context.Context, url string, out any) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", a.Marketplace)
	req.Header.Set("Accept", "application/json")

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		return json.Unmarshal(b, out)
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("browse api: http %d", resp.StatusCode)
	}
}

// --- Browse API wire shapes ---

type apiPrice struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

func (p apiPrice) num() float64 {
	v, _ := strconv.ParseFloat(p.Value, 64)
	return v
}

type apiItemSummary struct {
	ItemID          string     `json:"itemId"`
	LegacyItemID    string     `json:"legacyItemId"`
	Title           string     `json:"title"`
	Price           apiPrice   `json:"price"`
	Condition       string     `json:"condition"`
	BuyingOptions   []string   `json:"buyingOptions"`
	ItemWebURL      string     `json:"itemWebUrl"`
	Image           *apiImage  `json:"image"`
	Seller          *apiSeller `json:"seller"`
	ShippingOptions []struct {
		ShippingCost apiPrice `json:"shippingCost"`
	} `json:"shippingOptions"`
}

type apiImage struct {
	ImageURL string `json:"imageUrl"`
}

type apiSeller struct {
	Username      string `json:"username"`
	FeedbackScore int    `json:"feedbackScore"`
	FeedbackPct   string `json:"feedbackPercentage"`
}

// SearchItems runs a keyword search through the Browse API and maps the results
// to Listing records.
func (a *apiClient) SearchItems(ctx context.Context, query string, limit int) ([]*Listing, error) {
	if limit <= 0 || limit > 200 {
		limit = defaultPageSize
	}
	u := fmt.Sprintf("%s/item_summary/search?q=%s&limit=%d",
		a.browseBase, url.QueryEscape(query), limit)
	var resp struct {
		ItemSummaries []apiItemSummary `json:"itemSummaries"`
	}
	if err := a.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	var out []*Listing
	for _, s := range resp.ItemSummaries {
		out = append(out, s.toListing())
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s apiItemSummary) toListing() *Listing {
	id := s.LegacyItemID
	if id == "" {
		id = s.ItemID
	}
	l := &Listing{
		ID:        id,
		Title:     s.Title,
		Price:     s.Price.num(),
		Currency:  s.Price.Currency,
		Condition: s.Condition,
		URL:       s.ItemWebURL,
	}
	if l.URL == "" && id != "" {
		l.URL = BaseURL + "/itm/" + id
	}
	l.Buying = apiBuying(s.BuyingOptions)
	if s.Image != nil {
		l.Thumbnail = s.Image.ImageURL
	}
	if s.Seller != nil {
		l.Seller = s.Seller.Username
	}
	if len(s.ShippingOptions) > 0 {
		if c := s.ShippingOptions[0].ShippingCost; c.num() == 0 {
			l.Shipping = "Free"
		} else {
			l.Shipping = c.Currency + " " + c.Value
		}
	}
	return l
}

// GetItem fetches one item by its legacy (web) id and maps it to an Item record.
func (a *apiClient) GetItem(ctx context.Context, id string) (*Item, error) {
	u := fmt.Sprintf("%s/item/get_item_by_legacy_id?legacy_item_id=%s",
		a.browseBase, url.QueryEscape(id))
	var s struct {
		LegacyItemID     string     `json:"legacyItemId"`
		Title            string     `json:"title"`
		Subtitle         string     `json:"subtitle"`
		Price            apiPrice   `json:"price"`
		Condition        string     `json:"condition"`
		BuyingOptions    []string   `json:"buyingOptions"`
		ItemWebURL       string     `json:"itemWebUrl"`
		Image            *apiImage  `json:"image"`
		AdditionalImages []apiImage `json:"additionalImages"`
		Seller           *apiSeller `json:"seller"`
		ItemLocation     *struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"itemLocation"`
		CategoryPath          string `json:"categoryPath"`
		EstimatedAvailability *struct {
			EstimatedAvailableQuantity int `json:"estimatedAvailableQuantity"`
			EstimatedSoldQuantity      int `json:"estimatedSoldQuantity"`
		} `json:"estimatedAvailabilities"`
	}
	if err := a.getJSON(ctx, u, &s); err != nil {
		return nil, err
	}
	itemID := s.LegacyItemID
	if itemID == "" {
		itemID = id
	}
	it := &Item{
		ID:        itemID,
		Title:     s.Title,
		Subtitle:  s.Subtitle,
		Price:     s.Price.num(),
		Currency:  s.Price.Currency,
		Condition: s.Condition,
		Buying:    apiBuying(s.BuyingOptions),
		Category:  lastPath(s.CategoryPath),
		URL:       s.ItemWebURL,
	}
	if it.URL == "" {
		it.URL = BaseURL + "/itm/" + itemID
	}
	if s.Image != nil {
		it.Image = s.Image.ImageURL
		it.Images = append(it.Images, s.Image.ImageURL)
	}
	for _, img := range s.AdditionalImages {
		if img.ImageURL != "" {
			it.Images = append(it.Images, img.ImageURL)
		}
	}
	if s.Seller != nil {
		it.Seller = s.Seller.Username
		it.SellerScore = s.Seller.FeedbackScore
		it.SellerRating, _ = strconv.ParseFloat(s.Seller.FeedbackPct, 64)
	}
	if s.ItemLocation != nil {
		it.Location = squish(s.ItemLocation.City + " " + s.ItemLocation.Country)
	}
	if s.EstimatedAvailability != nil {
		it.Available = s.EstimatedAvailability.EstimatedAvailableQuantity
		it.Sold = s.EstimatedAvailability.EstimatedSoldQuantity
	}
	return it, nil
}

// apiBuying reduces the Browse buyingOptions list to the record's buying field.
func apiBuying(opts []string) string {
	for _, o := range opts {
		switch o {
		case "AUCTION":
			return "auction"
		case "FIXED_PRICE":
			return "fixed"
		case "BEST_OFFER":
			return "best_offer"
		}
	}
	return ""
}

// lastPath returns the leaf of a "A|B|C" category path.
func lastPath(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "|")
	return parts[len(parts)-1]
}
