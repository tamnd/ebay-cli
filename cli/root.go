// Package cli assembles the ebay command tree from the ebay domain on top of the
// any-cli/kit framework. Every read command is declared once as a kit operation
// in the ebay package, so the CLI, the HTTP API (ebay serve), and the MCP server
// (ebay mcp) all derive from one registry.
package cli

import (
	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/ebay-cli/ebay"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// builder holds the domain-global flags while the app is assembled, then folds
// them onto the resolved config in finalize.
type builder struct {
	userAgent   string
	marketplace string
	cacheTTL    string
	refresh     bool
}

// NewApp assembles the kit App: the ebay domain installs the client factory and
// the operations, this package adds the global flags and the version command,
// and kit provides the CLI, API, and MCP surfaces.
//
// To add a command, declare it in ebay/domain.go with kit.Handle and it appears
// here automatically. Reach for app.AddCommand only for a verb that does not fit
// the emit-records shape, the way version does below.
func NewApp() *kit.App {
	b := &builder{}
	id := ebay.Identity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(ebay.Defaults))
	app.GlobalFlags(b.globals)
	app.Finalize(b.finalize)

	ebay.Domain{}.Register(app)
	app.AddCommand(newVersionCmd())
	return app
}

func (b *builder) globals(f *kit.FlagSet) {
	f.StringVar(&b.userAgent, "user-agent", ebay.DefaultUserAgent, "User-Agent sent with each request")
	f.StringVar(&b.marketplace, "marketplace", "", "Browse API marketplace id (default EBAY_US)")
	f.StringVar(&b.cacheTTL, "cache-ttl", ebay.DefaultCacheTTL.String(), "how long a cached response stays fresh")
	f.BoolVar(&b.refresh, "refresh", false, "fetch fresh copies and rewrite the cache, ignoring any hit")
}

func (b *builder) finalize(c *kit.Config) {
	if c.Extra == nil {
		c.Extra = map[string]string{}
	}
	if b.userAgent != "" {
		c.Extra["user-agent"] = b.userAgent
	}
	if b.marketplace != "" {
		c.Extra["marketplace"] = b.marketplace
	}
	if b.cacheTTL != "" {
		c.Extra["cache-ttl"] = b.cacheTTL
	}
	if b.refresh {
		c.Extra["refresh"] = "true"
	}
}
