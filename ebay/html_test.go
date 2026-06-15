package ebay

import "testing"

func TestParsePrice(t *testing.T) {
	cases := []struct {
		in   string
		val  float64
		curr string
	}{
		{"$1,299.00", 1299, "USD"},
		{"US $19.99", 19.99, "USD"},
		{"£5", 5, "GBP"},
		{"€10", 10, "EUR"},
		{"$19.99 to $29.99", 19.99, "USD"}, // a range takes its low end
		{"15,103,947.37 VND", 15103947.37, "VND"},
		{"19.99 USD", 19.99, "USD"},
		{"Free shipping", 0, ""},
		{"", 0, ""},
	}
	for _, c := range cases {
		v, curr := parsePrice(c.in)
		if v != c.val || curr != c.curr {
			t.Errorf("parsePrice(%q) = %v %q, want %v %q", c.in, v, curr, c.val, c.curr)
		}
	}
}

func TestParseHuman(t *testing.T) {
	cases := []struct {
		in  string
		out int
	}{
		{"78K", 78000},
		{"1.2M", 1200000},
		{"768K", 768000},
		{"4,567", 4567},
		{"42", 42},
		{"3B", 3000000000},
		{"none", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := parseHuman(c.in); got != c.out {
			t.Errorf("parseHuman(%q) = %d, want %d", c.in, got, c.out)
		}
	}
}

func TestItemIDFrom(t *testing.T) {
	cases := []struct {
		in string
		id string
	}{
		{"https://www.ebay.com/itm/123456789012", "123456789012"},
		{"/itm/Apple-iPhone/285912345678", "285912345678"},
		{"/itm/285912345678?_trkparms=abc", "285912345678"},
		{"/help/home", ""},
	}
	for _, c := range cases {
		if got := itemIDFrom(c.in); got != c.id {
			t.Errorf("itemIDFrom(%q) = %q, want %q", c.in, got, c.id)
		}
	}
}
