package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/goforj/godump"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/lib"
	"golang.org/x/net/html"
)

func mpGetArticlePageData(queries *db.Queries, ctx context.Context, articleID int64) (mpdArticlePageData, error) {

	td := mpdArticlePageData{}

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

	alreadyRead, toRead, err := mpGetArticlesByFeedID(queries, fa.FeedID, ctx)
	if err != nil {
		return td, err
	}
	td.ArticlesRead = alreadyRead
	td.ArticlesToRead = toRead

	td.ClickableParagraphCount = fa.ArticleClickableParagraphCount

	td.PageContent = fa.ArticleContent

	enrichedHTML, err := mpEnrichHTMLOutputForDisplay(td.PageContent, fa.FeedID, articleID)
	if err != nil {
		return td, err
	}

	td.PageContent = enrichedHTML
	td.IsCache = true

	mns, err := mpGetComments(queries, ctx, articleID, -1)

	td.CommentsTemplateData = mns

	return td, nil

}

func mpUpdateComments(queries *db.Queries, ctx context.Context, noteText string, articleID int64, paragraphID int64) (mpdCommentsData, error) {

	mns := mpdCommentsData{}

	fmt.Printf("Does a note already exist - comment id: %v - note:%s\n", paragraphID, noteText)

	_, err := queries.SelectCommentsByArticleIDAndRelatedParagraphID(
		ctx, db.SelectCommentsByArticleIDAndRelatedParagraphIDParams{
			ArticleID:          articleID,
			RelatedParagraphID: paragraphID,
		},
	)

	//  If a note doesn't exist to update we INSERT a new one
	if err == sql.ErrNoRows {
		fmt.Println("No!")
		fmt.Printf("Creating a new note - paragraphID: %v - note:%s and returning all the notes\n", paragraphID, noteText)

		_, err := queries.InsertAndReturnComment(
			ctx,
			db.InsertAndReturnCommentParams{
				CommentText:        noteText,
				ArticleID:          articleID,
				RelatedParagraphID: paragraphID,
			},
		)
		if err != nil {
			return mns, err
		}

		return mpGetComments(queries, ctx, articleID, paragraphID)
	}

	if err != nil {
		return mns, err
	}

	fmt.Println("Yes!")
	fmt.Printf("Updating an existing note - paragraph id: %v - note:%s and returning all the notes\n", paragraphID, noteText)

	err = queries.UpdateCommentByArticleIDAndRelatedParagraphID(
		ctx,
		db.UpdateCommentByArticleIDAndRelatedParagraphIDParams{
			CommentText:        noteText,
			ArticleID:          articleID,
			RelatedParagraphID: paragraphID,
		},
	)
	if err != nil {
		return mns, err
	}

	return mpGetComments(queries, ctx, articleID, paragraphID)

}

func mpGetComments(queries *db.Queries, ctx context.Context, articleID int64, paragraphID int64) (mpdCommentsData, error) {

	mns := mpdCommentsData{}

	// CLARIFY!!!! if this is -1 then its the page render call
	fmt.Printf("Selecting note state: %v\n", paragraphID)
	mns.NoteToEdit = paragraphID

	article, err := queries.SelectArticleByID(ctx, articleID)
	if err != nil {
		return mns, err
	}
	mns.TotalPotentialCommentsCount = article.ClickableParagraphCount
	mns.ArticleID = article.ID

	notes, err := queries.SelectCommentsByArticleID(ctx, articleID)
	if err != nil {
		return mns, err
	}

	getParagraphID := func(n db.Comment) int64 {
		return n.RelatedParagraphID
	}

	notesMap := lib.SliceToMap(notes, getParagraphID)
	mns.Comments = notesMap

	return mns, nil
}

func mpSetArticleLike(queries *db.Queries, starredValue int64, articleID int64, ctx context.Context) error {

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

func mpGetArticlesByFeedID(queries *db.Queries, feedID int64, ctx context.Context) (alreadyRead []mpdFeedsArticle, toRead []mpdFeedsArticle, err error) {

	allArticles, err := queries.SelectArticlesByFeedID(ctx, feedID)
	if err != nil {
		return alreadyRead, toRead, err
	}

	for _, row := range allArticles {

		a := mpdFeedsArticle{
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

func mpEnrichHTMLOutputForDisplay(htmlStr string, _ int64, articleID int64) (string, error) {

	addDataAttributes := func(doc *html.Node) *html.Node {

		var walk func(*html.Node)

		count := 0

		walk = func(n *html.Node) {

			count++

			if n.Type == html.ElementNode {

				var paragraphID string

				for _, attr := range n.Attr {
					if attr.Key == "data-paragraph-id" {
						paragraphID = attr.Val
						break
					}
				}

				if paragraphID != "" {
					n.Attr = append(n.Attr, html.Attribute{
						Key: "data-on:click",
						Val: fmt.Sprintf("@get('/article/%v/comment/%v/write')", articleID, paragraphID),
					})
				}

			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}

		}

		walk(doc)

		return doc

	}

	removeOuterHTMLShell := func(doc *html.Node) (*html.Node, error) {

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

	htmlNodes, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", err
	}

	htmlNodes = addDataAttributes(htmlNodes)

	htmlNodes, err = removeOuterHTMLShell(htmlNodes)
	if err != nil {
		return "", err
	}

	transformed, err := _stringifyHTML(htmlNodes)
	if err != nil {
		return "", err
	}

	return transformed, nil

}

func mpEnrichArticles(queries *db.Queries, ctx context.Context, articles []db.SelectArticlesByFeedIDWithLimitRow) ([]mpdEnrichedArticle, error) {

	ea := []mpdEnrichedArticle{}
	a := mpdEnrichedArticle{}

	for i := range articles {
		if articles[i].ArticleContent != "" {
			enrichedContent, err := mpEnrichHTMLOutputForDisplay(
				articles[i].ArticleContent,
				0,
				articles[i].ArticleID,
			)

			if err != nil {
				return ea, err
			}

			articles[i].ArticleContent = enrichedContent
		}

		a.Article = articles[i]
		comments, err := queries.SelectCommentsByArticleID(ctx, articles[i].ArticleID)
		if err != nil {
			return ea, err
		}

		getParagraphID := func(n db.Comment) int64 {
			return n.RelatedParagraphID
		}

		commentsMap := lib.SliceToMap(comments, getParagraphID)
		a.CommentsData.Comments = commentsMap
		a.CommentsData.ArticleID = a.Article.ArticleID
		a.CommentsData.TotalPotentialCommentsCount = int64(len(comments))
		a.CommentsData.ShowTextArea = false

		ea = append(ea, a)

	}

	return ea, nil

}

func mpGetFeedUpdates(queries *db.Queries, ctx context.Context) (int64, error) {

	feeds, err := queries.SelectAllFeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("get feeds: %w", err)
	}

	parser := gofeed.NewParser()
	parser.Client = &http.Client{
		Timeout: 10 * time.Second,
	}

	var articlesInserted = 0
	for _, feed := range feeds {
		goFeed, err := parser.ParseURL(fmt.Sprintf("%s/feed/", feed.Url))
		if err != nil {
			return 0, fmt.Errorf("parse feed %s: %w", feed.Url, err)
		}

		if goFeed == nil {
			continue
		}

		for _, feedItem := range goFeed.Items {
			MpHTMLProcessingPipeline(queries, ctx, feedItem, feed)
			articlesInserted++
		}
	}

	_, err = queries.InsertAndReturnFeedsCallData(ctx, db.InsertAndReturnFeedsCallDataParams{
		RunType:         "user",
		ArticlesCreated: int64(articlesInserted),
	})

	if err != nil {
		return 0, fmt.Errorf("error inserting log call")
	}

	return int64(len(feeds)), nil
}

/*
*
This is called on generate or from the app when the user calls update
*
*/
func MpHTMLProcessingPipeline(queries *db.Queries, ctx context.Context, feedItem *gofeed.Item, feed db.Feed) (int64, error) {

	description, err := html.Parse(strings.NewReader(feedItem.Description))
	if err != nil {
		return 0, err
	}

	_sanitizeHTMLInput(description)

	sanitisedDesc, err := _stringifyHTML(description)
	if err != nil {
		return 0, err
	}

	scps := mpdPageScrapeParams{
		Link:           feedItem.Link,
		Container:      feed.CssSelContainer,
		ClipStartPoint: feed.CssSelStart,
		ClipEndPoint:   feed.CssSelStop,
		Strategy:       feed.HtmlExtractionStrategy,
	}

	rawHtml, err := _scrapeSiteHTML(scps)
	if err != nil {
		return 0, err
	}

	processedHtml, paragraphCount, err := _processScrapedHTML(rawHtml)
	if err != nil {
		log.Fatalf("error getting site html: %v", err)
	}

	now := time.Now()

	_, err = queries.InsertArticle(ctx, db.InsertArticleParams{
		FeedID:                  feed.ID,
		Title:                   feedItem.Title,
		Link:                    feedItem.Link,
		Published:               _getFeedItemDate(feedItem),
		ClickableParagraphCount: paragraphCount,
		ArticleContent:          processedHtml,
		ScrapedHtml:             rawHtml,
		DateFound:               &now,
		Summary:                 sanitisedDesc,
		Read:                    0,
		Starred:                 0,
	})
	if err != nil {
		return 0, fmt.Errorf("insert article: %w", err)
	}

	return paragraphCount, nil

}

type mpdParagraph struct {
	Text string
}

func mpGetHTMLChunksByIndex(htmlInput string, index int) ([]mpdParagraph, error) {

	allChunks := [][]mpdParagraph{}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlInput))
	if err != nil {
		return []mpdParagraph{}, err
	}

	allNodes := doc.Find("body > p, body > blockquote")

	// is the index out of range
	if index*3 >= allNodes.Length() {
		return []mpdParagraph{}, errors.New("index out of range!")
	}

	for i := 0; i < allNodes.Length(); i += 3 {

		group := allNodes.Slice(i, min(i+3, allNodes.Length()))
		chunks := []mpdParagraph{}

		group.Each(func(_ int, s *goquery.Selection) {
			chunks = append(chunks, mpdParagraph{Text: s.Text()})
		})
		allChunks = append(allChunks, chunks)

	}

	godump.Dump("all", allChunks)

	return allChunks[index], nil
}

func _scrapeSiteHTML(feed mpdPageScrapeParams) (string, error) {

	godump.Dump("feed", feed)

	pageHtmlContent := ""

	c := colly.NewCollector()

	c.OnHTML(feed.Container, func(h *colly.HTMLElement) {
		pageHtmlContent = _extractHTMLRange(h.DOM, feed.ClipStartPoint, feed.ClipEndPoint)
	})

	if err := c.Visit(feed.Link); err != nil {
		return "", fmt.Errorf("error using colly to visit page: %v - %v", feed.Link, err)
	}

	return pageHtmlContent, nil

}

func _extractHTMLRange(container *goquery.Selection, startSelector, stopSelector string) string {

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

func _processScrapedHTML(input string) (string, int64, error) {

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

	_sanitizeHTMLInput(doc)

	paragraphCount := enrichHTML(doc)

	stringifiedHTML, err := _stringifyHTML(doc)
	if err != nil {
		return "", 0, err
	}

	return stringifiedHTML, paragraphCount, nil
}

func _stringifyHTML(doc *html.Node) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func _sanitizeHTMLInput(doc *html.Node) {

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

func _getFeedItemDate(item *gofeed.Item) *time.Time {
	if item.PublishedParsed != nil {
		return item.PublishedParsed
	}

	if item.UpdatedParsed != nil {
		return item.UpdatedParsed
	}

	return nil
}

// Added to keep hold of a reference to godump
func DUMMY_godump(message string, val any) {
	godump.Dump(message, val)
}

type mpdSidebarLink struct {
	Name   string
	Link   string
	Unread int64
	FeedId int
}

type mpdArticlePageData struct {
	FeedID                  int64
	PageTitle               string
	ArticlesRead            []mpdFeedsArticle
	ArticlesToRead          []mpdFeedsArticle
	FeedTitle               string
	FeedUrl                 string
	Link                    string
	PageContent             string
	ArticleId               int64
	IsCache                 bool
	StarValue               int64
	Sidebar                 []mpdSidebarLink
	ArticlePublished        string
	ArticleRead             int64
	MarginNotes             map[int64]db.Comment
	ClickableParagraphCount int64
	CommentsTemplateData    mpdCommentsData
}

func (ae mpdArticlePageData) ArticleHasBeenRead() bool {
	return lib.IntToBool(ae.ArticleRead)
}

type mpdFeedSummary struct {
	Name          string
	ArticleCount  int64
	FeedID        int64
	PageID        int64
	LinksRequired int64
	Articles      []mpdEnrichedArticle //[]db.SelectArticlesByFeedIDWithLimitRow
	ShowArticles  bool
}

const layoutISO = "2006-01-02"

type mpdCommentsData struct {
	ShowTextArea                bool
	ArticleID                   int64
	NoteToEdit                  int64
	TotalPotentialCommentsCount int64
	Comments                    map[int64]db.Comment
}

type mpdPageScrapeParams struct {
	Link           string
	Container      string
	ClipStartPoint string
	ClipEndPoint   string
	Strategy       string
}

type mpdCreateFeedSignals struct {
	Title                  string `json:"feed-name"`
	FeedUrl                string `json:"feed-url"`
	CSSSelectorContainer   string `json:"css-sel-container"`
	CSSSelectorStart       string `json:"css-sel-start"`
	CSSSelectorStop        string `json:"css-sel-stop"`
	HTMLExtractionStrategy string `json:"html-extraction-strategy"`
}

type mpdEnrichedArticle struct {
	Article      db.SelectArticlesByFeedIDWithLimitRow
	CommentsData mpdCommentsData
}

type mpdFeedsArticle struct {
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

func (a mpdFeedsArticle) FullName() string {
	return a.FeedTitle + " - " + a.Title
}

func (a mpdFeedsArticle) ScrubbedSummary() template.HTML {
	p := bluemonday.UGCPolicy()
	return template.HTML(p.Sanitize(a.Summary))
}

func (a mpdFeedsArticle) PublishedDate() string {

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
