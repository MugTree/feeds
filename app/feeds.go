package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/lib"
	"github.com/starfederation/datastar/sdk/go/datastar"
	"golang.org/x/net/html"
)

func feedsGetArticlePageState(queries *db.Queries, ctx context.Context, articleID int64, feedID int64) (ArticlePageState, error) {

	td := ArticlePageState{}

	sidebar, err := feedsGetSideBarTemplateData(queries, ctx)
	if err != nil {
		return td, errors.New("error getting sidebar template data: " + err.Error())
	}
	td.Sidebar = sidebar

	fa, err := queries.SelectFeedAndArticletByArticleID(ctx, articleID)
	if err != nil {
		return td, errors.New("error getting article data: " + err.Error())
	}

	td.PageTitle = fa.ArticleTitle
	td.FeedTitle = fa.FeedTitle
	td.FeedUrl = fa.FeedUrl
	td.Link = fa.ArticleLink
	td.ArticleId = fa.ArticleID
	td.FeedID = fa.FeedID
	td.ArticleRead = fa.ArticleRead

	td.StarValue = fa.ArticleStars
	td.ArticlePublished = fa.ArticlePublished.Format(layoutISO)

	alreadyRead, toRead, err := feedsGetArticlesByFeedID(queries, feedID, ctx)
	if err != nil {
		return td, err
	}
	td.ArticlesRead = alreadyRead
	td.ArticlesToRead = toRead

	hasContent, preCachedHTML, clickableBlocksCount, err := feedsGetArticleContentIfCached(queries, td.Link, fa.ArticleID, ctx)
	if err != nil {
		return td, err
	}

	if hasContent {

		enrichedHTMLForUser, err := feedsEnrichHTMLOutput(preCachedHTML)
		if err != nil {
			return td, err
		}

		td.PageContent = enrichedHTMLForUser
		td.ClickableBlockCount = clickableBlocksCount
		td.IsCache = true

		notes, err := queries.SelectMarginNotesByArticleID(ctx, articleID)
		if err != nil {
			return td, err
		}

		// these need to be used as a lookup in the template
		notesMap := lib.SliceToMap(notes, func(n db.MarginNote) int64 {
			return n.ID
		})

		td.MarginNotes = notesMap
		return td, nil
	}

	newHTML, clickableBlocksCount, err := feedsNetRetrieveArticleHTML(queries, fa, ctx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return td, err
		}
		return td, err
	}

	processedHTML, clickableBlocksCount, err := feedsProcessScrapedHTML(newHTML)
	if err != nil {
		return td, err
	}

	newlyCached, err := queries.InsertAndReturnCachedArticle(ctx, db.InsertAndReturnCachedArticleParams{
		ArticleID:           articleID,
		Link:                td.Link,
		ArticleContent:      sql.NullString{String: processedHTML, Valid: true},
		ClickableBlockCount: clickableBlocksCount,
	})

	if err != nil {
		return td, err
	}

	enrichedHTMLForUser, err := feedsEnrichHTMLOutput(newlyCached.ArticleContent.String)

	td.PageContent = enrichedHTMLForUser
	td.ClickableBlockCount = newlyCached.ClickableBlockCount

	return td, nil
}

func feedsGetSideBarTemplateData(queries *db.Queries, ctx context.Context) ([]feedsSidebarLink, error) {

	items := []feedsSidebarLink{}
	data, err := queries.SelectSideBarData(ctx)
	if err != nil {
		return items, err
	}

	for _, row := range data {
		items = append(items, feedsSidebarLink{
			Name:   row.FeedTitle,
			Link:   fmt.Sprintf("/feed/%v/view", row.FeedID),
			Unread: (row.TotalArticles - row.ArticlesRead),
		})
	}

	return items, nil
}

func feedsSetArticleLike(queries *db.Queries, starredValue int64, articleID int64, ctx context.Context) error {

	updatedValue := func(currentValue int64) int64 {
		if currentValue == 3 {
			return 0
		}
		return currentValue + 1
	}(starredValue)

	err := queries.UpdateArticleSetStarredValue(ctx,
		db.UpdateArticleSetStarredValueParams{
			Starred: int64(updatedValue),
			ID:      articleID},
	)
	if err != nil {
		return err
	}

	return nil
}

func feedsGetHomePageArticleSelections(queries *db.Queries, ctx context.Context) (latest []feedsArticle, starred []feedsArticle, err error) {

	latest5Articles, err := queries.SelectLatest5Articles(ctx)
	if err != nil {
		return latest, starred, err
	}

	for _, row := range latest5Articles {
		latest = append(latest, feedsArticle{
			Id:        row.ID,
			FeedId:    row.FeedID,
			Title:     row.Title,
			Link:      row.Link,
			Published: row.Published.Format(layoutISO),
			DateFound: row.DateFound.Format(layoutISO),
			Summary:   row.Summary,
			Read:      lib.IntToBool(row.Read),
			Liked:     row.Starred,
			FeedTitle: row.FeedTitle,
		})
	}

	starredArticles, err := queries.SelectLatest5StarredArticles(ctx)

	for _, row := range starredArticles {
		starred = append(starred, feedsArticle{
			Id:        row.ID,
			FeedId:    row.FeedID,
			Title:     row.Title,
			Link:      row.Link,
			Published: row.Published.Format(layoutISO),
			DateFound: row.Published.Format(layoutISO),
			Summary:   row.Summary,
			Read:      lib.IntToBool(row.Read),
			Liked:     row.Starred,
			FeedTitle: row.FeedTitle,
		})
	}

	fmt.Printf("starred: %v", len(starred))

	return latest, starred, err

}

func feedsGetArticlesByFeedID(queries *db.Queries, feedID int64, ctx context.Context) (alreadyRead []feedsArticle, toRead []feedsArticle, err error) {

	allArticles, err := queries.SelectArticlesByFeedID(ctx, feedID)
	if err != nil {
		return alreadyRead, toRead, err
	}

	for _, row := range allArticles {

		a := feedsArticle{
			Id:        row.ID,
			FeedId:    row.FeedID,
			Title:     row.Title,
			Link:      row.Link,
			Published: row.Published.Format(layoutISO),
			DateFound: row.DateFound.Format(layoutISO),
			Summary:   row.Summary,
			Read:      lib.IntToBool(row.Read),
			Liked:     row.Starred,
			FeedTitle: row.FeedTitle,
		}

		if a.Read {
			alreadyRead = append(alreadyRead, a)
			continue
		}

		toRead = append(toRead, a)
	}

	return alreadyRead, toRead, nil
}

func feedsGetArticleContentIfCached(queries *db.Queries, articleLink string, _ int64, ctx context.Context) (bool, string, int64, error) {

	var clickableBlocks int64 = 0

	lc, err := queries.SelectCachedArticleByLink(ctx, articleLink)
	if err == nil {

		// parsedHTML, err := html.Parse(strings.NewReader(lc.ArticleContent.String))
		// if err != nil {
		// 	return false, "", 0, err
		// }

		// article, err := feedsRemoveOuterHTMLShell(parsedHTML)
		// if err != nil {
		// 	return false, "", clickableBlocks, err
		// }

		// articleStr, err := feedsStringifyHTML(article)
		// if err != nil {
		// 	return false, "", clickableBlocks, err
		// }

		clickableBlocks := lc.ClickableBlockCount

		return true, lc.ArticleContent.String, clickableBlocks, nil
	}
	if err == sql.ErrNoRows {
		return false, "", clickableBlocks, nil
	}

	return false, "", clickableBlocks, err

}

func feedsExtractHTMLRangeFlat(container *goquery.Selection, startSelector, stopSelector string) string {

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

func feedsGetFeedItemDate(item *gofeed.Item) *time.Time {
	if item.PublishedParsed != nil {
		return item.PublishedParsed
	}

	if item.UpdatedParsed != nil {
		return item.UpdatedParsed
	}

	return nil
}

/* returns the fully processed information plus some data about the processing */
func feedsProcessScrapedHTML(input string) (string, int64, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", 0, err
	}

	_feedsSanitizeHTMLInput(doc)

	clickableBlockCount := _feedsEnrichHTMLInput(doc)

	stringifiedHTML, err := feedsStringifyHTML(doc)
	if err != nil {
		return "", 0, err
	}

	return stringifiedHTML, clickableBlockCount, nil
}

/* before data is passed to the front end we add some additional properties for interactivity*/
func feedsEnrichHTMLOutput(htmlStr string) (string, error) {

	addDataAtrtibutes := func(doc *html.Node) *html.Node {

		var walk func(*html.Node)

		walk = func(n *html.Node) {

			if n.Type == html.ElementNode {

				var blockID string

				for _, attr := range n.Attr {
					if attr.Key == "data-block-id" {
						blockID = attr.Val
						break
					}
				}

				if blockID != "" {
					n.Attr = append(n.Attr, html.Attribute{
						Key: "data-on:click",
						Val: datastar.GetSSE("/url/%s", blockID),
					})
				}

				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}

			}

		}

		walk(doc)

		return doc

	}

	htmlNodes, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	htmlNodes = addDataAtrtibutes(htmlNodes)
	htmlNodes, err = _feedsRemoveOuterHTMLShell(htmlNodes)
	if err != nil {
		return "", err
	}

	transformed, err := feedsStringifyHTML(htmlNodes)
	if err != nil {
		return "", err
	}

	return transformed, nil

}

// article for the article, div for the desc from feeds
func feedsStringifyHTML(doc *html.Node) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

/* remove the outer shell so we can place it into the users HTML without breaking the layout */
func _feedsRemoveOuterHTMLShell(doc *html.Node) (*html.Node, error) {

	var walk func(*html.Node)

	walk = func(n *html.Node) {
		// if doc != nil {
		// 	return
		// }

		if n.Type == html.ElementNode && n.Data == "body" {
			doc = n
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	if doc == nil {
		return nil, fmt.Errorf("body element not found")
	}

	article := &html.Node{
		Type: html.ElementNode,
		Data: "article",
	}

	// Move every child from <body> into <article>.
	for doc.FirstChild != nil {
		child := doc.FirstChild
		doc.RemoveChild(child)
		article.AppendChild(child)
	}

	return article, nil
}

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

/* adding some properties to the HTML coming that we are ingesting */
func _feedsEnrichHTMLInput(doc *html.Node) int64 {

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
						Key: "data-block-id",
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

type feedsArticle struct {
	Id        int64  `json:"id" db:"id"`
	FeedId    int64  `json:"feed_id" db:"feed_id"`
	Title     string `json:"title" db:"title"`
	Link      string `json:"link" db:"link"`
	Published string `json:"published" db:"published"`
	DateFound string `json:"date_found" db:"date_found"`
	Summary   string `json:"summary" db:"summary"`
	Read      bool   `json:"read" db:"read"`
	Liked     int64  `json:"starred" db:"starred"`
	FeedTitle string `json:"feed_title" db:"feed_title"`
}

func (a feedsArticle) FullName() string {
	return a.FeedTitle + " - " + a.Title
}

func (a feedsArticle) ScrubbedSummary() template.HTML {
	p := bluemonday.UGCPolicy()
	return template.HTML(p.Sanitize(a.Summary))
}

func (a feedsArticle) PublishedDate() string {

	d, err := time.Parse(time.RFC1123Z, a.Published)
	if err != nil {
		fmt.Printf("time parse issue: %v", err)
		return ""
	}

	day := d.Day()
	month := d.Format("January")
	year := d.Year()

	suffix := "th"
	if day%10 == 1 && day != 11 {
		suffix = "st"
	} else if day%10 == 2 && day != 12 {
		suffix = "nd"
	} else if day%10 == 3 && day != 13 {
		suffix = "rd"
	}

	return fmt.Sprintf("%d%s %s %d", day, suffix, month, year)
}

type feedsSidebarLink struct {
	Name   string
	Link   string
	Unread int64
	FeedId int
}

type ArticlePageState struct {
	FeedID              int64
	PageTitle           string
	ArticlesRead        []feedsArticle
	ArticlesToRead      []feedsArticle
	FeedTitle           string
	FeedUrl             string
	Link                string
	PageContent         string
	ArticleId           int64
	IsCache             bool
	StarValue           int64
	Sidebar             []feedsSidebarLink
	ArticlePublished    string
	ArticleRead         int64
	MarginNotes         map[int64]db.MarginNote
	ClickableBlockCount int64
}

func (ae ArticlePageState) ArticleHasBeenRead() bool {
	return lib.IntToBool(ae.ArticleRead)
}

type FeedFormTemplateData struct {
	ButtonText string
	UrlAction  string
	Feed       db.Feed
}

type ArticleStatus struct{ HasBeenRead bool }

const layoutISO = "2006-01-02"
