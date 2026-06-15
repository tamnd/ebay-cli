package ebay

import "strings"

// ids.go resolves a reference to a (kind, id) pair and builds canonical URLs,
// all offline. It backs `ebay ref id` and `ebay ref url`, and the Resolver the
// ant host calls to turn an ebay:// URI into the right command.

// Classify reads a reference (a URL, a path, or a bare id) and reports what it
// points at. Kind is one of item, seller, category, or unknown.
func Classify(ref string) Ref {
	in := strings.TrimSpace(ref)
	r := Ref{Input: in, Kind: "unknown"}
	if in == "" {
		return r
	}

	// A bare item id: a run of 9 to 13 digits, the shape of an eBay item number.
	if isItemID(in) {
		r.Kind, r.ID = "item", in
		r.URL = URLFor(r.Kind, r.ID)
		return r
	}

	segs := splitSegs(refPath(in))
	if len(segs) == 0 {
		return r
	}

	switch segs[0] {
	case "itm":
		// /itm/<id> or /itm/<slug>/<id>: the id is the last all-digit segment.
		if id := lastDigits(segs[1:]); id != "" {
			r.Kind, r.ID = "item", id
		}
	case "usr", "str":
		if len(segs) >= 2 {
			r.Kind, r.ID = "seller", segs[1]
		}
	case "b":
		// /b/<slug>/<catid>/bn_<node> or /b/<slug>/<catid>: the category id is
		// the last all-digit segment that is not the bn_ node.
		if id := lastDigits(segs[1:]); id != "" {
			r.Kind, r.ID = "category", id
		}
	case "sch":
		// /sch/i.html?_sacat=<catid>: a category search.
		if id := query(in, "_sacat"); id != "" && id != "0" {
			r.Kind, r.ID = "category", id
		}
	}
	if r.Kind != "unknown" {
		r.URL = URLFor(r.Kind, r.ID)
	}
	return r
}

// URLFor builds the canonical eBay URL for a (kind, id) pair.
func URLFor(kind, id string) string {
	switch kind {
	case "item":
		return BaseURL + "/itm/" + id
	case "seller":
		return BaseURL + "/usr/" + id
	case "category":
		return BaseURL + "/b/" + id
	default:
		return ""
	}
}

// refPath reduces a reference to a site path: a full URL loses scheme and host,
// a bare path is returned trimmed.
func refPath(ref string) string {
	if i := strings.Index(ref, "://"); i >= 0 {
		rest := ref[i+3:]
		if s := strings.IndexByte(rest, '/'); s >= 0 {
			rest = rest[s:]
		} else {
			return "/"
		}
		ref = rest
	}
	// Drop a query string before splitting into path segments.
	if q := strings.IndexByte(ref, '?'); q >= 0 {
		ref = ref[:q]
	}
	if !strings.HasPrefix(ref, "/") {
		ref = "/" + ref
	}
	return ref
}

// query returns the value of a query parameter in a URL, or "" when absent.
func query(ref, key string) string {
	q := strings.IndexByte(ref, '?')
	if q < 0 {
		return ""
	}
	for _, pair := range strings.Split(ref[q+1:], "&") {
		k, v, ok := strings.Cut(pair, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}

func splitSegs(path string) []string {
	var out []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// lastDigits returns the last all-digit segment that is not a bn_ browse node,
// the shape of an item id in /itm/ and a category id in /b/.
func lastDigits(segs []string) string {
	id := ""
	for _, s := range segs {
		if isDigits(s) {
			id = s
		}
	}
	return id
}

// isItemID reports whether s is a bare eBay item number: 9 to 13 digits.
func isItemID(s string) bool {
	if len(s) < 9 || len(s) > 13 {
		return false
	}
	return isDigits(s)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
