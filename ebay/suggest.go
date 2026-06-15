package ebay

import (
	"context"
	"encoding/json"
	"net/url"
)

// suggest.go reads the autocomplete host, the most reliable surface in the tool:
// a small JSON host the search box calls as you type, not behind the bot wall.
// The reply is the OpenSearch array ["<prefix>", ["term", "term", ...]].

// Suggest returns search-autocomplete terms for a typed prefix.
func (c *Client) Suggest(ctx context.Context, prefix string, limit int) ([]*Suggestion, error) {
	base := c.SuggestURL
	if base == "" {
		base = autosugURL
	}
	u := base + "?kwd=" + url.QueryEscape(prefix) + "&sId=0&fmt=osr"
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	// The array is heterogeneous: a string then an array of strings.
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 2 {
		return nil, nil
	}
	var terms []string
	if err := json.Unmarshal(raw[1], &terms); err != nil {
		return nil, nil
	}
	var out []*Suggestion
	for _, t := range terms {
		out = append(out, &Suggestion{Query: prefix, Term: t})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}
