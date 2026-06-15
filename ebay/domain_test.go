package ebay

import (
	"errors"
	"testing"

	"github.com/tamnd/any-cli/kit/errs"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		in   string
		kind string
		id   string
		url  string
	}{
		{"123456789012", "item", "123456789012", "https://www.ebay.com/itm/123456789012"},
		{"https://www.ebay.com/itm/123456789012", "item", "123456789012", "https://www.ebay.com/itm/123456789012"},
		{"https://www.ebay.com/itm/Apple-iPhone-13/285912345678", "item", "285912345678", "https://www.ebay.com/itm/285912345678"},
		{"https://www.ebay.com/usr/techbargains", "seller", "techbargains", "https://www.ebay.com/usr/techbargains"},
		{"https://www.ebay.com/str/techbargainsstore", "seller", "techbargainsstore", "https://www.ebay.com/usr/techbargainsstore"},
		{"https://www.ebay.com/b/Cell-Phones-Smartphones/9355/bn_320094", "category", "9355", "https://www.ebay.com/b/9355"},
		{"https://www.ebay.com/b/Cell-Phones/9355", "category", "9355", "https://www.ebay.com/b/9355"},
		{"https://www.ebay.com/sch/i.html?_sacat=9355&_nkw=phone", "category", "9355", "https://www.ebay.com/b/9355"},
		{"/usr/someseller", "seller", "someseller", "https://www.ebay.com/usr/someseller"},
		{"", "unknown", "", ""},
		{"https://www.ebay.com/help/home", "unknown", "", ""},
		{"12345", "unknown", "", ""}, // too short for an item id
	}
	for _, c := range cases {
		got := Classify(c.in)
		if got.Kind != c.kind || got.ID != c.id || got.URL != c.url {
			t.Errorf("Classify(%q) = {%q %q %q}, want {%q %q %q}",
				c.in, got.Kind, got.ID, got.URL, c.kind, c.id, c.url)
		}
	}
}

// TestURLForRoundTrip checks that re-classifying a built URL recovers the same
// (kind, id) for the kinds that have a canonical URL.
func TestURLForRoundTrip(t *testing.T) {
	cases := []struct{ kind, id string }{
		{"item", "123456789012"},
		{"seller", "techbargains"},
		{"category", "9355"},
	}
	for _, c := range cases {
		u := URLFor(c.kind, c.id)
		if u == "" {
			t.Errorf("URLFor(%q,%q) empty", c.kind, c.id)
			continue
		}
		got := Classify(u)
		if got.Kind != c.kind || got.ID != c.id {
			t.Errorf("round-trip %q: Classify(%q) = {%q %q}, want {%q %q}",
				c.kind, u, got.Kind, got.ID, c.kind, c.id)
		}
	}
}

func TestURLForUnknown(t *testing.T) {
	if u := URLFor("nonsense", "x"); u != "" {
		t.Errorf("URLFor of an unknown kind should be empty, got %q", u)
	}
}

func TestMapErr(t *testing.T) {
	cases := []struct {
		err  error
		kind errs.Kind
	}{
		{ErrNotFound, errs.KindNotFound},
		{ErrRateLimited, errs.KindRateLimited},
		{ErrBlocked, errs.KindNeedAuth},
	}
	for _, c := range cases {
		got := mapErr(c.err)
		if errs.KindOf(got) != c.kind {
			t.Errorf("mapErr(%v) kind = %v, want %v", c.err, errs.KindOf(got), c.kind)
		}
	}
	if mapErr(nil) != nil {
		t.Error("mapErr(nil) should be nil")
	}
	plain := errors.New("boom")
	if got := mapErr(plain); got != plain {
		t.Errorf("mapErr passes an unmapped error through unchanged, got %v", got)
	}
}

func TestDomainClassify(t *testing.T) {
	d := Domain{}
	kind, id, err := d.Classify("https://www.ebay.com/itm/123456789012")
	if err != nil {
		t.Fatal(err)
	}
	if kind != "item" || id != "123456789012" {
		t.Errorf("Domain.Classify = {%q %q}", kind, id)
	}
	if _, _, err := d.Classify("https://www.ebay.com/help/home"); err == nil {
		t.Error("Domain.Classify of an unknown reference should error")
	}
}

func TestDomainLocate(t *testing.T) {
	d := Domain{}
	u, err := d.Locate("category", "9355")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://www.ebay.com/b/9355" {
		t.Errorf("Locate = %q", u)
	}
	if _, err := d.Locate("nonsense", "x"); err == nil {
		t.Error("Locate of an unknown kind should error")
	}
}

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "ebay" {
		t.Errorf("scheme = %q", info.Scheme)
	}
	want := map[string]bool{"www.ebay.com": true, "ebay.com": true, "m.ebay.com": true}
	for _, h := range info.Hosts {
		delete(want, h)
	}
	if len(want) != 0 {
		t.Errorf("missing hosts: %v", want)
	}
}
