package ebay

import "time"

// Host is the eBay hostname this client builds page URLs from and the host the
// URI driver in domain.go claims.
const Host = "www.ebay.com"

// BaseURL is the root every item, seller, and category URL is built from.
const BaseURL = "https://" + Host

// autosugURL is the autocomplete host the search box calls as you type. It is a
// small JSON host, not behind the bot wall, so `ebay suggest` is the most
// reliable command in the tool.
const autosugURL = "https://autosug.ebay.com/autosug"

// The opt-in Browse API. eBay walls the item page and keyword search from
// datacenter IPs, so when EBAY_CLIENT_ID and EBAY_CLIENT_SECRET are set the
// walled commands fall back to this documented backend instead of failing.
const (
	apiBase       = "https://api.ebay.com"
	tokenEndpoint = apiBase + "/identity/v1/oauth2/token"
	browseBase    = apiBase + "/buy/browse/v1"
	apiScope      = "https://api.ebay.com/oauth/api_scope"
	defaultMarket = "EBAY_US"
)

// DefaultUserAgent is sent with every page request. eBay serves its public
// pages to a normal browser; a browser User-Agent is what keeps a logged-out
// reader looking like one. Override it with --user-agent.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Defaults for the polite client.
const (
	// DefaultDelay is the minimum gap between requests. eBay is touchier than a
	// plain API, so a one-second pace reads steadily without leaning on it.
	DefaultDelay    = 1 * time.Second
	DefaultRetries  = 3
	DefaultTimeout  = 30 * time.Second
	DefaultCacheTTL = 24 * time.Hour

	// defaultPageSize is the items asked for per listing page. 60 is the size
	// the web grids use.
	defaultPageSize = 60
)

// Config carries the knobs the client reads. It is built from the kit framework
// config in ClientFromConfig, so a --rate or --timeout on the command line and
// the same value resolved by a host both land here.
type Config struct {
	UserAgent   string
	Marketplace string

	// ClientID and ClientSecret enable the Browse API fallback for the walled
	// surfaces (item, search). Empty leaves the fallback off; the walled
	// commands then report the bot wall rather than guessing.
	ClientID     string
	ClientSecret string

	// Delay is the minimum gap between requests. Zero means no pacing.
	Delay   time.Duration
	Retries int
	Timeout time.Duration

	// BaseURL is the site root. Empty uses the public site; tests point it at an
	// httptest server.
	BaseURL string

	// CacheDir is where responses are cached. Empty disables the cache, as does
	// NoCache.
	CacheDir string
	CacheTTL time.Duration
	NoCache  bool
	// Refresh fetches fresh copies and rewrites the cache, ignoring any hit.
	Refresh bool
}

// DefaultConfig returns the baseline configuration: a browser User-Agent, the
// EBAY_US marketplace, a one-second pace, three retries, a 30s timeout, and a
// one-day cache.
func DefaultConfig() Config {
	return Config{
		UserAgent:   DefaultUserAgent,
		Marketplace: defaultMarket,
		Delay:       DefaultDelay,
		Retries:     DefaultRetries,
		Timeout:     DefaultTimeout,
		BaseURL:     BaseURL,
		CacheTTL:    DefaultCacheTTL,
	}
}
