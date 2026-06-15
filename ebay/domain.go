package ebay

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes ebay as a kit Domain: a driver that a multi-domain host
// (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/ebay-cli/ebay"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// ebay:// URIs by routing to the operations Register installs. The same Domain
// also builds the standalone ebay binary (see cli.NewApp), so the binary and a
// host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the ebay driver. It carries no state; the per-run client is built by
// the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:   "ebay",
		Hosts:    []string{Host, "ebay.com", "m.ebay.com"},
		Identity: Identity(),
	}
}

// Identity is the fixed description of the ebay CLI, shared by the domain and the
// standalone composition root so help and version read the same everywhere.
func Identity() kit.Identity {
	return kit.Identity{
		Binary: "ebay",
		Short:  "Read public eBay listings, items, sellers, categories, and deals into structured records",
		Long: `ebay reads public eBay data the way a logged-out browser does:
keyword search, a single item, a seller and their listings, a category
and the items in it, the current deals, and search autocomplete. The
category, seller, deals, and autocomplete surfaces read from any
network; the item page and keyword search are walled from datacenter
IPs by eBay's bot manager, so those are best-effort and fall back to
the Browse API when EBAY_CLIENT_ID and EBAY_CLIENT_SECRET are set.
There is no API key needed for the core, no login, and nothing to run
alongside it. It returns records as a table, JSON, JSONL, CSV, TSV, or
URLs, and serves the same operations over HTTP and MCP.

ebay is an independent tool and is not affiliated with eBay.`,
		Site: BaseURL,
		Repo: "https://github.com/tamnd/ebay-cli",
	}
}

// Register installs the client factory and every operation onto app. A resolver
// op (Single) names its own record type and answers `ant get`; a List op
// enumerates a parent resource's members and answers `ant ls`.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	app.CommandGroup("read", "Read public eBay data")
	app.CommandGroup("seller", "Read a seller and their listings")
	app.CommandGroup("category", "Read a category, its items, and its children")
	app.CommandGroup("ref", "Resolve references to ids and URLs (offline)")

	// Top-level reads.
	kit.Handle(app, kit.OpMeta{
		Name: "search", Group: "read", List: true,
		Summary: "Search listings by keyword",
		URIType: "item",
		Args:    []kit.Arg{{Name: "query", Help: "search keywords"}},
	}, search)

	kit.Handle(app, kit.OpMeta{
		Name: "item", Group: "read", Single: true,
		Summary: "Show one item by id",
		URIType: "item", Resolver: true,
		Args: []kit.Arg{{Name: "id", Help: "item id or /itm/ URL"}},
	}, getItem)

	kit.Handle(app, kit.OpMeta{
		Name: "deals", Group: "read", List: true,
		Summary: "The current deals",
		URIType: "item",
	}, deals)

	kit.Handle(app, kit.OpMeta{
		Name: "suggest", Group: "read", List: true,
		Summary: "Search autocomplete suggestions",
		Args:    []kit.Arg{{Name: "prefix", Help: "the typed prefix"}},
	}, suggest)

	// Seller: metadata, listings.
	kit.Handle(app, kit.OpMeta{
		Name: "show", Parent: "seller", Single: true,
		Summary: "Show a seller's profile",
		URIType: "seller", Resolver: true,
		Args: []kit.Arg{{Name: "username", Help: "seller username, store, or URL"}},
	}, getSeller)

	kit.Handle(app, kit.OpMeta{
		Name: "listings", Parent: "seller", List: true,
		Summary: "List a seller's active listings",
		URIType: "item",
		Args:    []kit.Arg{{Name: "username", Help: "seller username, store, or URL"}},
	}, sellerListings)

	// Category: metadata, items, children.
	kit.Handle(app, kit.OpMeta{
		Name: "show", Parent: "category", Single: true,
		Summary: "Show a category's metadata",
		URIType: "category", Resolver: true,
		Args: []kit.Arg{{Name: "id", Help: "category id or /b/ URL"}},
	}, getCategory)

	kit.Handle(app, kit.OpMeta{
		Name: "browse", Parent: "category", List: true,
		Summary: "List the items in a category",
		URIType: "item",
		Args:    []kit.Arg{{Name: "id", Help: "category id or /b/ URL"}},
	}, categoryBrowse)

	kit.Handle(app, kit.OpMeta{
		Name: "tree", Parent: "category", List: true,
		Summary: "List a category's child categories",
		URIType: "category",
		Args:    []kit.Arg{{Name: "id", Help: "category id or /b/ URL"}},
	}, categoryTree)

	// Reference tools (offline).
	kit.Handle(app, kit.OpMeta{
		Name: "id", Parent: "ref", Single: true,
		Summary: "Classify a reference into its (kind, id)",
		Args:    []kit.Arg{{Name: "ref", Help: "any eBay URL, path, or id"}},
	}, classifyRef)

	kit.Handle(app, kit.OpMeta{
		Name: "url", Parent: "ref", Single: true,
		Summary: "Build the canonical URL for a (kind, id)",
		Args: []kit.Arg{
			{Name: "kind", Help: "item, seller, or category"},
			{Name: "id", Help: "the id for that kind"},
		},
	}, buildURL)
}

// newClient builds the client from the host-resolved config, so a host and the
// standalone binary pace and identify themselves the same way.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	return ClientFromConfig(cfg), nil
}

// ClientFromConfig maps the framework config onto an ebay.Config and returns a
// client. The Browse credentials are read from the environment, not flags, so
// they never land in shell history.
func ClientFromConfig(cfg kit.Config) *Client {
	ec := DefaultConfig()
	if cfg.Rate > 0 {
		ec.Delay = cfg.Rate
	}
	if cfg.Retries >= 0 {
		ec.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		ec.Timeout = cfg.Timeout
	}
	if ua := cfg.Extra["user-agent"]; ua != "" {
		ec.UserAgent = ua
	} else if cfg.UserAgent != "" {
		ec.UserAgent = cfg.UserAgent
	}
	if m := cfg.Extra["marketplace"]; m != "" {
		ec.Marketplace = m
	} else if m := os.Getenv("EBAY_MARKETPLACE"); m != "" {
		ec.Marketplace = m
	}
	ec.ClientID = os.Getenv("EBAY_CLIENT_ID")
	ec.ClientSecret = os.Getenv("EBAY_CLIENT_SECRET")
	ec.CacheDir = cfg.CacheDir
	ec.NoCache = cfg.NoCache
	if ttl := cfg.Extra["cache-ttl"]; ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			ec.CacheTTL = d
		}
	}
	ec.Refresh = cfg.Extra["refresh"] == "true"
	return NewClient(ec)
}

// Defaults seeds the framework baseline with ebay's own values, so an unset
// --rate or --timeout uses the ebay default rather than the generic kit one. It
// is passed to kit.New via kit.WithDefaults.
func Defaults(c *kit.Config) {
	def := DefaultConfig()
	c.Rate = def.Delay
	c.Retries = def.Retries
	c.Timeout = def.Timeout
	c.UserAgent = def.UserAgent
}

// Classify turns any accepted input into the canonical (type, id), so `ant
// resolve` and `ant url` touch no network.
func (Domain) Classify(input string) (uriType, id string, err error) {
	r := Classify(input)
	if r.Kind == "unknown" {
		return "", "", errs.Usage("unrecognized ebay reference: %q", input)
	}
	return r.Kind, r.ID, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	u := URLFor(uriType, id)
	if u == "" {
		return "", errs.Usage("ebay has no resource type %q", uriType)
	}
	return u, nil
}

// mapErr translates a library error into a kit error so the exit code matches the
// rest of the fleet: a missing entity reads as "not found" (exit 6), a throttle
// as "rate limited" (exit 5), and the bot wall as "need auth" (exit 4).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	case errors.Is(err, ErrRateLimited):
		return errs.RateLimited("%s", err.Error())
	case errors.Is(err, ErrBlocked):
		return errs.NeedAuth("%s", err.Error())
	default:
		return err
	}
}

// limitOr returns the operator's --limit when set, else the command's own
// default fetch count.
func limitOr(limit, def int) int {
	if limit > 0 {
		return limit
	}
	return def
}
