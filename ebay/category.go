package ebay

import (
	"bytes"
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// category.go reads the category browse page (/b/), one of the reliable-core
// surfaces eBay serves anonymously from any network. It reads the JSON-LD first
// (the breadcrumb trail and the per-item aggregate ratings) and the HTML cards
// for the listings, folding the ratings onto the cards.

// categoryURL builds the short browse URL eBay redirects to the slugged page. It
// accepts a bare category id or a full /b/ URL.
func (c *Client) categoryURL(ref string) string {
	r := Classify(ref)
	if r.Kind == "category" {
		return c.BaseURL + "/b/" + r.ID
	}
	return c.BaseURL + "/b/" + strings.Trim(ref, "/")
}

// CategoryBrowse returns the listings in a category.
func (c *Client) CategoryBrowse(ctx context.Context, ref string, limit int) ([]*Listing, error) {
	body, err := c.get(ctx, c.categoryURL(ref))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	listings := parseCards(doc, limit)
	rate := productRatings(parseJSONLD(doc))
	for _, l := range listings {
		if r, ok := rate[l.ID]; ok {
			l.Rating = r.value
			l.Reviews = r.reviews
		}
	}
	return listings, nil
}

// GetCategory returns a category's metadata: its name and its parent, from the
// breadcrumb trail on the browse page.
func (c *Client) GetCategory(ctx context.Context, ref string) (*Category, error) {
	r := Classify(ref)
	id := r.ID
	if r.Kind != "category" {
		id = strings.Trim(ref, "/")
	}
	body, err := c.get(ctx, c.categoryURL(ref))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	cat := &Category{ID: id, URL: BaseURL + "/b/" + id}
	trail := breadcrumb(parseJSONLD(doc))
	if len(trail) > 0 {
		cat.Name = trail[len(trail)-1]
		if len(trail) > 1 {
			cat.Parent = trail[len(trail)-2]
		}
	}
	if cat.Name == "" {
		// Fall back to the page heading.
		cat.Name = squish(firstText(doc.Selection, "h1.b-title", "h1"))
	}
	cat.Node = categoryNode(body)
	return cat, nil
}

// CategoryTree returns the child categories of a node, read from the left-rail
// category links the browse page renders.
func (c *Client) CategoryTree(ctx context.Context, ref string, limit int) ([]*Category, error) {
	body, err := c.get(ctx, c.categoryURL(ref))
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var out []*Category
	seen := map[string]bool{}
	doc.Find(`a[href*="/b/"]`).EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")
		id := categoryIDFrom(href)
		if id == "" || seen[id] {
			return true
		}
		name := squish(a.Text())
		if name == "" {
			return true
		}
		seen[id] = true
		out = append(out, &Category{ID: id, Name: name, URL: BaseURL + "/b/" + id})
		return !(limit > 0 && len(out) >= limit)
	})
	return out, nil
}

// categoryIDFrom extracts the numeric category id from a /b/ href.
func categoryIDFrom(href string) string {
	segs := splitSegs(refPath(href))
	if len(segs) == 0 || segs[0] != "b" {
		return ""
	}
	return lastDigits(segs[1:])
}

// categoryNode pulls the bn_<digits> browse-node id out of the page when present.
func categoryNode(body []byte) string {
	i := bytes.Index(body, []byte("bn_"))
	if i < 0 {
		return ""
	}
	j := i + 3
	for j < len(body) && body[j] >= '0' && body[j] <= '9' {
		j++
	}
	if j == i+3 {
		return ""
	}
	return string(body[i:j])
}
