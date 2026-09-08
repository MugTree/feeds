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

func getArticlePageData(queries *db.Queries, ctx context.Context, articleID int64) (ArticlePageTemplateData, error) {

	td := ArticlePageTemplateData{}

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

	alreadyRead, toRead, err := getArticlesByFeedID(queries, fa.FeedID, ctx)
	if err != nil {
		return td, err
	}
	td.ArticlesRead = alreadyRead
	td.ArticlesToRead = toRead

	td.ClickableParagraphCount = fa.ArticleClickableParagraphCount

	td.PageContent = fa.ArticleContent

	enrichedHTML, err := enrichHTMLOutput(td.PageContent, fa.FeedID, articleID)
	if err != nil {
		return td, err
	}

	td.PageContent = enrichedHTML
	td.IsCache = true

	mns, err := getComments(queries, ctx, articleID, -1)

	td.CommentsTemplateData = mns

	return td, nil

}

/* This needs to update or insert a specific margin note and then return all the margin notes */
func updateComments(queries *db.Queries, ctx context.Context, noteText string, articleID int64, paragraphID int64) (CommentsTemplateData, error) {

	mns := CommentsTemplateData{}

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

		return getComments(queries, ctx, articleID, paragraphID)
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

	return getComments(queries, ctx, articleID, paragraphID)

}

func getComments(queries *db.Queries, ctx context.Context, articleID int64, paragraphID int64) (CommentsTemplateData, error) {

	mns := CommentsTemplateData{}

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

func setArticleLike(queries *db.Queries, starredValue int64, articleID int64, ctx context.Context) error {

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

func getArticlesByFeedID(queries *db.Queries, feedID int64, ctx context.Context) (alreadyRead []feedsArticle, toRead []feedsArticle, err error) {

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

/* before data is passed to the front end we add some additional properties for interactivity*/
func enrichHTMLOutput(htmlStr string, _ int64, articleID int64) (string, error) {

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

	transformed, err := stringifyHTML(htmlNodes)
	if err != nil {
		return "", err
	}

	return transformed, nil

}

// this needs to return something slightly different
func enrichArticles(queries *db.Queries, ctx context.Context, articles []db.SelectArticlesByFeedIDWithLimitRow) ([]EnrichedArticle, error) {

	ea := []EnrichedArticle{}
	a := EnrichedArticle{}

	for i := range articles {
		if articles[i].ArticleContent != "" {
			enrichedContent, err := enrichHTMLOutput(
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

func stringifyHTML(doc *html.Node) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

/* adding some properties to the HTML coming that we are ingesting */

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

	var articlesInserted = 0

	for _, feed := range feeds {

		goFeed, err := parser.ParseURL(fmt.Sprintf("%s/feed/", feed.Url))
		if err != nil {
			return 0, fmt.Errorf("parse feed %s: %w", feed.Url, err)
		}

		if goFeed == nil {
			continue
		}

		//#REFACTOR -  there's a good amount of duplication here with the generation script
		// ./app/db/seed/generate.go

		for _, item := range goFeed.Items {

			now := time.Now()

			description, err := html.Parse(strings.NewReader(item.Description))
			if err != nil {
				return 0, err
			}

			_feedsSanitizeHTMLInput(description)

			output, err := StringifyHTML(description)
			if err != nil {
				return 0, err
			}

			html, err := ScrapeSiteHTML(PageScrapeParams{
				Link:           item.Link,
				Container:      feed.CssSelContainer,
				ClipStartPoint: feed.CssSelStart,
				ClipEndPoint:   feed.CssSelStop,
			})
			if err != nil {
				return 0, err
			}

			processed, paragraphCount, err := ProcessScrapedHTML(html)
			if err != nil {
				log.Fatalf("error getting site html: %v", err)
			}

			_, err = queries.InsertArticle(ctx, db.InsertArticleParams{
				FeedID:                  feed.ID,
				Title:                   item.Title,
				Link:                    item.Link,
				Published:               feedsGetFeedItemDate(item),
				ClickableParagraphCount: paragraphCount,
				ArticleContent:          processed,
				DateFound:               &now,
				Summary:                 output,
				Read:                    0,
				Starred:                 0,
			})
			if err != nil {
				return 0, fmt.Errorf("insert article: %w", err)
			}
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

func feedsGetFeedItemDate(item *gofeed.Item) *time.Time {
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

type EnrichedArticle struct {
	Article      db.SelectArticlesByFeedIDWithLimitRow
	CommentsData CommentsTemplateData
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

type ArticlePageTemplateData struct {
	FeedID                  int64
	PageTitle               string
	ArticlesRead            []feedsArticle
	ArticlesToRead          []feedsArticle
	FeedTitle               string
	FeedUrl                 string
	Link                    string
	PageContent             string
	ArticleId               int64
	IsCache                 bool
	StarValue               int64
	Sidebar                 []feedsSidebarLink
	ArticlePublished        string
	ArticleRead             int64
	MarginNotes             map[int64]db.Comment
	ClickableParagraphCount int64
	CommentsTemplateData    CommentsTemplateData
}

func (ae ArticlePageTemplateData) ArticleHasBeenRead() bool {
	return lib.IntToBool(ae.ArticleRead)
}

type ArticleStatus struct {
	HasBeenRead                  bool
	HasScrolledToBottomOfArticle bool
}

type CommentsTemplateData struct {
	ShowTextArea                bool
	ArticleID                   int64
	NoteToEdit                  int64
	TotalPotentialCommentsCount int64
	Comments                    map[int64]db.Comment
}

type FeedSummary struct {
	Name          string
	ArticleCount  int64
	FeedID        int64
	PageID        int64
	LinksRequired int64
	Articles      []EnrichedArticle //[]db.SelectArticlesByFeedIDWithLimitRow
	ShowArticles  bool
}

type NewHomePageTemplateData struct {
	FeedMeta       []FeedSummary
	ArticlesByFeed map[string][]db.SelectArticlesByFeedIDWithLimitRow
}

const layoutISO = "2006-01-02"

type FrontPageSignals struct {
	ArticlesOpen []int64 `json:"articlesOpen"`
	FeedsOpen    []int64 `json:"feedsOpen"`
}

type PageScrapeParams struct {
	Link           string
	Container      string
	ClipStartPoint string
	ClipEndPoint   string
	Strategy       string
}

type FeedCreateUpdateSignals struct {
	Title                  string `json:"feed-name"`
	FeedUrl                string `json:"feed-url"`
	CSSSelectorContainer   string `json:"css-sel-container"`
	CSSSelectorStart       string `json:"css-sel-start"`
	CSSSelectorStop        string `json:"css-sel-stop"`
	HTMLExtractionStrategy string `json:"html-extraction-strategy"`
}
