package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
	"golang.org/x/net/html"
)

// func FAKE_getAnnotations() []feedsAnnotation {
// 	return []feedsAnnotation{{
// 		ID:        1,
// 		StartData: feedsAnnotationData{Path: []int64{0, 4}, Offset: 8},
// 		EndData:   feedsAnnotationData{Path: []int64{0, 4}, Offset: 8},
// 	}, {
// 		ID:        2,
// 		StartData: feedsAnnotationData{Path: []int64{0, 6}, Offset: 10},
// 		EndData:   feedsAnnotationData{Path: []int64{3, 4}, Offset: 9},
// 	}}

// }

func feedsGetArticlePageTemplateData(queries *db.Queries, ctx context.Context, articleID int64, feedID int64) (ArticlePageTemplateData, error) {

	td := ArticlePageTemplateData{}

	sidebar, err := feedsGetSideBarTemplateData(queries, ctx)
	if err != nil {
		return td, err
	}
	td.Sidebar = sidebar

	row, err := queries.SelectFeedAndArticletByArticleID(ctx, articleID)
	if err != nil {
		return td, err
	}

	td.PageTitle = row.ArticleTitle
	td.FeedTitle = row.FeedTitle
	td.FeedUrl = row.FeedUrl
	td.Link = row.ArticleLink
	td.ArticleId = row.ArticleID
	td.FeedID = row.FeedID
	td.StarValue = row.ArticleStars
	td.ArticlePublished = row.ArticlePublished.Format(layoutISO)
	//td.Annotations = FAKE_getAnnotations()

	alreadyRead, toRead, err := feedsGetArticlesByFeedID(queries, feedID, ctx)
	if err != nil {
		return td, err
	}
	td.ArticlesRead = alreadyRead
	td.ArticlesToRead = toRead

	hasContent, cachedContent, err := feedsIsArticleCached(queries, td.Link, ctx)
	if err != nil {
		return td, err
	}

	if hasContent {
		td.PageContent = cachedContent
		td.IsCache = true
	} else {

		newContent, err := feedsGetArticleHTMLFromWeb(queries, row, ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return td, err
			}
			return td, err
		}
		td.PageContent = newContent
	}

	// fmt.Printf("FeedTitle: %v\n", td.FeedTitle)
	// fmt.Printf("AlreadyRead: %v\n", len(td.AlreadyRead))
	// fmt.Printf("ToRead: %v\n", len(td.ToRead))

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
			Read:      int64ToBool(row.Read),
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
			Read:      int64ToBool(row.Read),
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
			Read:      int64ToBool(row.Read),
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

func feedsIsArticleCached(queries *db.Queries, articleLink string, ctx context.Context) (bool, string, error) {

	lc, err := queries.SelectCachedArticleByLink(ctx, articleLink)
	if err == nil {
		return true, lc.ArticleContent.String, nil
	}
	if err == sql.ErrNoRows {
		return false, "", nil
	}

	return false, "", err

}

func feedsGetArticleHTMLFromWeb(queries *db.Queries, afd db.SelectFeedAndArticletByArticleIDRow, ctx context.Context) (string, error) {

	pageHtmlContent := ""

	type extractionParams struct {
		Container      string
		ClipStartPoint string
		ClipEndPoint   string
	}

	ep := extractionParams{}
	ep.Container = afd.FeedCssSelContainer.String

	switch afd.FeedHtmlExtractionStrategy.String {
	case "no-clip":
		break
	case "clip-start":
		ep.ClipStartPoint = afd.FeedCssSelStart.String
	case "clip-end":
		ep.ClipEndPoint = afd.FeedCssSelStop.String
	case "clip-between":
		ep.ClipStartPoint = afd.FeedCssSelStart.String
		ep.ClipEndPoint = afd.FeedCssSelStop.String
	}

	//TODO - need to add some timeout values here really
	c := colly.NewCollector()

	c.OnHTML(ep.Container, func(h *colly.HTMLElement) {
		pageHtmlContent = feedsExtractHTMLRangeFlat(h.DOM, ep.ClipStartPoint, ep.ClipEndPoint)
	})

	if err := c.Visit(afd.ArticleLink); err != nil {
		return "", fmt.Errorf("error using colly to visit page: %v - %v", afd.ArticleLink, err)
	}

	sanitizedHtml, err := feedsAnnotationSanitizeHTMLForStorage(pageHtmlContent)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	err = html.Render(&output, sanitizedHtml)
	if err != nil {
		return "", err
	}

	insertErr := queries.InsertCachedArticle(ctx,
		db.InsertCachedArticleParams{
			ArticleID:      afd.ArticleID,
			Link:           afd.ArticleLink,
			ArticleContent: sql.NullString{String: output.String(), Valid: true},
		},
	)

	if insertErr != nil {
		return "", fmt.Errorf("error adding to article cache: %v", insertErr)
	}

	return "", nil
}

// func feedsHTMLArticleReplaceBodyTag(root *html.Node) error {

// 	body := feedsFindNodeInHTML(root, "body")
// 	if body == nil {
// 		return fmt.Errorf("body not found")
// 	}

// 	parent := body.Parent
// 	if parent == nil {
// 		return fmt.Errorf("body has no parent")
// 	}

// 	article := &html.Node{
// 		Type: html.ElementNode,
// 		Data: "article",
// 	}

// 	// move all children from body -> article
// 	for c := body.FirstChild; c != nil; {

// 		next := c.NextSibling

// 		body.RemoveChild(c)
// 		article.AppendChild(c)

// 		c = next
// 	}

// 	// replace body with article
// 	parent.InsertBefore(article, body)
// 	parent.RemoveChild(body)

// 	return nil
// }

// func feedsFindNodeInHTML(n *html.Node, tag string) *html.Node {

// 	var walk func(*html.Node) *html.Node

// 	walk = func(n *html.Node) *html.Node {

// 		if n.Type == html.ElementNode && n.Data == tag {
// 			return n
// 		}

// 		for c := n.FirstChild; c != nil; c = c.NextSibling {
// 			if res := walk(c); res != nil {
// 				return res
// 			}
// 		}

// 		return nil
// 	}

// 	return walk(n)
// }

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

func feedsGetFeedUpdates(queries *db.Queries, ctx context.Context) (int64, error) {

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

			sanitizedHtml, err := feedsAnnotationSanitizeHTMLForStorage(item.Description)
			if err != nil {
				return 0, err
			}

			var output strings.Builder
			err = html.Render(&output, sanitizedHtml)
			if err != nil {
				return 0, err
			}

			err = queries.InsertOrIgnoreArticle(ctx, db.InsertOrIgnoreArticleParams{
				FeedID:    feed.ID,
				Title:     item.Title,
				Link:      item.Link,
				Published: feedsGetFeedItemDate(item),
				DateFound: &now,
				Summary:   output.String(),
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

// type feedsArticlePlusFeed struct {
// 	ArticleID              int64
// 	ArticleLink            string
// 	ArticleTitle           string
// 	ArticleStarred         int64
// 	FeedID                 int64
// 	FeedTitle              string
// 	FeedUrl                string
// 	CssSelContainer        sql.NullString
// 	CssSelStart            sql.NullString
// 	CssSelStop             sql.NullString
// 	HtmlExtractionStrategy sql.NullString
// }

type feedsSidebarLink struct {
	Name   string
	Link   string
	Unread int64
	FeedId int
}

func int64ToBool(i int64) bool {
	if i == 0 {
		return false
	}
	return true
}

type ArticlePageTemplateData struct {
	FeedID           int64
	PageTitle        string
	ArticlesRead     []feedsArticle
	ArticlesToRead   []feedsArticle
	FeedTitle        string
	FeedUrl          string
	Link             string
	PageContent      string
	ArticleId        int64
	IsCache          bool
	StarValue        int64
	Sidebar          []feedsSidebarLink
	ArticlePublished string
	//Annotations      []feedsAnnotation
}

// type feedsUpdateParms struct {
// 	FeedId   int64
// 	PageType string
// }

type FeedFormTemplateData struct {
	ButtonText string
	UrlAction  string
	Feed       db.Feed
}

// type feedsAnnotation struct {
// 	ID        int64               `json:"id"`
// 	ArticleID int64               `json:"article_id"`
// 	StartData feedsAnnotationData `json:"start_data"`
// 	EndData   feedsAnnotationData `json:"end_data"`
// 	Snippet   string              `json:"snippet"`
// 	Note      string              `json:"note"`
// 	DateAdded string              `json:"date_added"`
// }

// type feedsAnnotationData struct {
// 	Path   []int64 `json:"path"`
// 	Offset int64   `json:"offset"`
// }

// type TextNode struct {
// 	Node  *html.Node
// 	Start int
// 	End   int
// }

const layoutISO = "2006-01-02"
