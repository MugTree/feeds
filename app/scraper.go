package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
	"golang.org/x/net/html"
)

type PageScrapeParams struct {
	Link           string
	Container      string
	ClipStartPoint string
	ClipEndPoint   string
	Strategy       string
}

func ScrapeSiteHTML(ep PageScrapeParams) (string, error) {

	pageHtmlContent := ""

	//ep.Container = afd.FeedCssSelContainer.String

	// switch wp.FeedHtmlExtractionStrategy.String {
	// case "no-clip":
	// 	break
	// case "clip-start":
	// 	ep.ClipStartPoint = afd.FeedCssSelStart.String
	// case "clip-end":
	// 	ep.ClipEndPoint = afd.FeedCssSelStop.String
	// case "clip-between":
	// 	ep.ClipStartPoint = afd.FeedCssSelStart.String
	// 	ep.ClipEndPoint = afd.FeedCssSelStop.String
	// }

	//TODO - need to add some timeout values here really
	c := colly.NewCollector()

	c.OnHTML(ep.Container, func(h *colly.HTMLElement) {
		pageHtmlContent = ExtractHTMLRangeFlat(h.DOM, ep.ClipStartPoint, ep.ClipEndPoint)
	})

	if err := c.Visit(ep.Link); err != nil {
		return "", fmt.Errorf("error using colly to visit page: %v - %v", ep.Link, err)
	}

	return pageHtmlContent, nil

}

func ExtractHTMLRangeFlat(container *goquery.Selection, startSelector, stopSelector string) string {

	var chunks []string
	started := startSelector == ""
	stopped := false

	container.Children().Each(func(i int, sel *goquery.Selection) {
		if stopped {
			return
		}

		if !started {
			if startSelector != "" && sel.Is(startSelector) {
				started = true
			} else {
				return
			}
		}

		if stopSelector != "" && sel.Is(stopSelector) {
			stopped = true
			return
		}

		if html, err := goquery.OuterHtml(sel); err == nil {
			// fmt.Println(html)
			// fmt.Println("---------------------------------")
			chunks = append(chunks, html)
		}
	})

	return strings.Join(chunks, "")
}

func ProcessScrapedHTML(input string) (string, int64, error) {

	enrichHTML := func(doc *html.Node) int64 {

		isBlockElement := func(tag string) bool {
			switch tag {
			case "p",
				// "h1",
				// "h2",
				// "h3",
				// "h4",
				// "div",
				"figure",
				"blockquote",
				"ul",
				"ol",
				"table":
				return true
			default:
				return false
			}
		}

		id := 0

		var walk func(*html.Node, bool)

		walk = func(n *html.Node, ancestorIsBlock bool) {

			for c := n.FirstChild; c != nil; c = c.NextSibling {

				childAncestorIsBlock := ancestorIsBlock

				if c.Type == html.ElementNode {

					tag := strings.ToLower(c.Data)

					if isBlockElement(tag) && !ancestorIsBlock {

						c.Attr = append(c.Attr, html.Attribute{
							Key: "data-paragraph-id",
							Val: strconv.Itoa(id),
						})

						id++
						childAncestorIsBlock = true
					}
				}

				if c.FirstChild != nil {
					walk(c, childAncestorIsBlock)
				}
			}
		}

		walk(doc, false)

		return int64(id)
	}

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", 0, err
	}

	_feedsSanitizeHTMLInput(doc)

	paragraphCount := enrichHTML(doc)

	stringifiedHTML, err := StringifyHTML(doc)
	if err != nil {
		return "", 0, err
	}

	return stringifiedHTML, paragraphCount, nil
}

// article for the article, div for the desc from feeds
func StringifyHTML(doc *html.Node) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

/* remove the outer shell so we can place it into the users HTML without breaking the layout */

/* most of the data that we scrape comes with a lot of stuff attached that we dont want*/
func _feedsSanitizeHTMLInput(doc *html.Node) {

	allowedAttrs := func(tag string) map[string]struct{} {
		switch tag {
		case "a":
			return map[string]struct{}{
				"href": {},
			}
		case "img":
			return map[string]struct{}{
				"src": {},
				"alt": {},
			}
		case "td", "th":
			return map[string]struct{}{
				"colspan": {},
				"rowspan": {},
			}
		default:
			return nil
		}
	}

	shouldRemoveElement := func(n *html.Node) bool {
		if n.Type != html.ElementNode {
			return false
		}

		switch strings.ToLower(n.Data) {
		case "script", "noscript", "style", "template":
			return true
		default:
			return false
		}
	}

	var clean func(*html.Node)

	clean = func(n *html.Node) {

		for c := n.FirstChild; c != nil; {

			next := c.NextSibling

			// Remove unwanted elements.
			if shouldRemoveElement(c) {
				n.RemoveChild(c)
				c = next
				continue
			}

			// Remove comments.
			if c.Type == html.CommentNode {
				n.RemoveChild(c)
				c = next
				continue
			}

			// Remove whitespace-only text nodes.
			if c.Type == html.TextNode &&
				strings.TrimSpace(c.Data) == "" {

				n.RemoveChild(c)
				c = next
				continue
			}

			if c.Type == html.ElementNode {

				tag := strings.ToLower(c.Data)

				// Strip unwanted attributes.
				allowed := allowedAttrs(tag)

				attrs := c.Attr[:0]
				for _, attr := range c.Attr {
					if _, ok := allowed[attr.Key]; ok {
						attrs = append(attrs, attr)
					}
				}
				c.Attr = attrs
			}

			// Recurse first so children are cleaned before
			// deciding whether this node is empty.
			if c.FirstChild != nil {
				clean(c)
			}

			// Remove empty elements.
			if c.Type == html.ElementNode &&
				len(c.Attr) == 0 &&
				c.FirstChild == nil {

				switch strings.ToLower(c.Data) {
				case "div", "span", "p":
					n.RemoveChild(c)
					c = next
					continue
				}
			}

			c = next
		}
	}

	clean(doc)
}

func GetFeedUpdates(queries *db.Queries, ctx context.Context) (int64, error) {

	feeds, err := queries.SelectAllFeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("get feeds: %w", err)
	}

	parser := gofeed.NewParser()
	parser.Client = &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, feed := range feeds {

		goFeed, err := parser.ParseURL(fmt.Sprintf("%s/feed/", feed.Url))
		if err != nil {
			return 0, fmt.Errorf("parse feed %s: %w", feed.Url, err)
		}

		if goFeed == nil {
			continue
		}

		for _, item := range goFeed.Items {

			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
			}

			now := time.Now()

			doc, err := html.Parse(strings.NewReader(item.Description))
			if err != nil {
				return 0, err
			}

			_feedsSanitizeHTMLInput(doc)

			output, err := StringifyHTML(doc)
			if err != nil {
				return 0, err
			}

			err = queries.InsertOrIgnoreArticle(ctx, db.InsertOrIgnoreArticleParams{
				FeedID:    feed.ID,
				Title:     item.Title,
				Link:      item.Link,
				Published: feedsGetFeedItemDate(item),
				DateFound: &now,
				Summary:   output,
				Read:      0,
				Starred:   0,
			})
			if err != nil {
				return 0, fmt.Errorf("insert article: %w", err)
			}
		}
	}

	return int64(len(feeds)), nil
}

func feedsGetFeedItemDate(item *gofeed.Item) *time.Time {
	if item.PublishedParsed != nil {
		return item.PublishedParsed
	}

	if item.UpdatedParsed != nil {
		return item.UpdatedParsed
	}

	return nil
}
