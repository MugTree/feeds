package app

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
)

func getSidebarData(queries *db.Queries, ctx context.Context) ([]SidebarLink, error) {

	items := []SidebarLink{}
	data, err := queries.GetSidebarData(ctx)
	if err != nil {
		return items, err
	}

	for _, v := range data {
		items = append(items, mapSidebarLinkFromSidebarDataRow(v))
	}

	return items, nil
}

func getHomepageArticles(queries *db.Queries, ctx context.Context) ([]Article, error) {

	articles := []Article{}
	latest5Articles, err := queries.GetLatest5Articles(ctx)
	if err != nil {
		return articles, err
	}

	for _, v := range latest5Articles {
		articles = append(articles, mapArticleFromLatest5ArticlesRow(v))
	}

	return articles, err

}

func getUnreadArticles(queries *db.Queries, feedID int64, ctx context.Context) ([]Article, error) {

	articles := []Article{}
	unreadArticles, err := queries.GetUnreadByFeedID(ctx, feedID)
	if err != nil {
		return articles, err
	}

	for _, v := range unreadArticles {
		articles = append(articles, mapArticleFromUnreadByFeedIDRow(v))
	}

	return articles, nil
}

func getArticleAndFeed(queries *db.Queries, articleID int64, ctx context.Context) (ArticleAndFeed, error) {

	afd := ArticleAndFeed{}

	fd, err := queries.GetFeedDataForArticleByArticleID(ctx, articleID)
	if err != nil {
		return afd, err
	}

	afd = mapArticleWithFeedDataFromArticleByArticleIDRow(fd)
	return afd, nil

}

func hasCachedContent(queries *db.Queries, articleLink string, ctx context.Context) (bool, string, error) {

	lc, err := queries.GetCachedByLink(ctx, articleLink)
	if err == nil {
		return true, lc.ArticleContent.String, nil
	}
	if err == sql.ErrNoRows {
		return false, "", nil
	}

	return false, "", err

}

func getArticleFromWeb(queries *db.Queries, afd ArticleAndFeed, ctx context.Context) (string, error) {

	pageHtmlContent := ""

	type extractionParams struct {
		Container      string
		ClipStartPoint string
		ClipEndPoint   string
	}

	ep := extractionParams{}
	ep.Container = afd.CssSelContainer.String

	switch afd.HtmlExtractionStrategy.String {
	case "no-clip":
		break
	case "clip-start":
		ep.ClipStartPoint = afd.CssSelStart.String
	case "clip-end":
		ep.ClipEndPoint = afd.CssSelStop.String
	case "clip-between":
		ep.ClipStartPoint = afd.CssSelStart.String
		ep.ClipEndPoint = afd.CssSelStop.String
	}

	//TODO - need to add some timeout values here really
	c := colly.NewCollector()

	c.OnHTML(ep.Container, func(h *colly.HTMLElement) {
		pageHtmlContent = ExtractHTMLRangeFlat(h.DOM, ep.ClipStartPoint, ep.ClipEndPoint)
	})

	if err := c.Visit(afd.ArticleLink); err != nil {
		return "", fmt.Errorf("error using colly to visit page: %v - %v", afd.ArticleLink, err)
	}

	insertErr := queries.AddToArticleCache(ctx,
		db.AddToArticleCacheParams{
			Link:           afd.ArticleLink,
			ArticleContent: sql.NullString{String: pageHtmlContent, Valid: true},
		},
	)

	if insertErr != nil {
		return "", fmt.Errorf("error adding to article cache: %v", insertErr)
	}

	return pageHtmlContent, nil
}

// func getFeedIndexData(feedId int64, queries *db.Queries, ctx context.Context) (FeedPageVM, error) {

// func getArticleData(articleId int64, feedId int64, queries *db.Queries, ctx context.Context) (ArticlePageVM, error) {

// 	vm := ArticlePageVM{}

// 	afd, err := getArticleWithFeedData(queries, articleId, ctx)
// 	if err != nil {
// 		return vm, err
// 	}

// 	var pageHtmlContent = ""

// 	lc, err := queries.GetCachedByLink(ctx, afd.ArticleLink)
// 	isCached := false

// 	if err == nil {
// 		isCached = true
// 		pageHtmlContent = lc.ArticleContent.String
// 	} else {

// 		if err == sql.ErrNoRows {

// 			type extractionParams struct {
// 				Container      string
// 				ClipStartPoint string
// 				ClipEndPoint   string
// 			}

// 			ep := extractionParams{}
// 			ep.Container = afd.CssSelContainer.String

// 			switch afd.HtmlExtractionStrategy.String {
// 			case "no-clip":
// 				break
// 			case "clip-start":
// 				ep.ClipStartPoint = afd.CssSelStart.String
// 			case "clip-end":
// 				ep.ClipEndPoint = afd.CssSelStop.String
// 			case "clip-between":
// 				ep.ClipStartPoint = afd.CssSelStart.String
// 				ep.ClipEndPoint = afd.CssSelStop.String
// 			}

// 			//TODO - need to add some timeout values here really
// 			c := colly.NewCollector()

// 			c.OnHTML(ep.Container, func(h *colly.HTMLElement) {
// 				pageHtmlContent = ExtractHTMLRangeFlat(h.DOM, ep.ClipStartPoint, ep.ClipEndPoint)
// 			})

// 			if err := c.Visit(afd.ArticleLink); err != nil {
// 				return vm, fmt.Errorf("error using colly to visit page: %v - %v", afd.ArticleLink, err)
// 			}

// 			insertErr := queries.AddToArticleCache(ctx,
// 				db.AddToArticleCacheParams{
// 					Link:           afd.ArticleLink,
// 					ArticleContent: sql.NullString{String: pageHtmlContent, Valid: true},
// 				},
// 			)

// 			if insertErr != nil {
// 				return vm, fmt.Errorf("error adding to article cache: %v", insertErr)
// 			}
// 		} else {
// 			return vm, fmt.Errorf("error querying article cache: %v", err)
// 		}
// 	}

// 	// get other page parts
// 	// --------------------------------------------------------
// 	sbd, err := getSidebarData(queries, ctx)
// 	if err != nil {
// 		return vm, fmt.Errorf("error getting side data: %v", err)
// 	}

// 	unread, err := queries.GetUnreadByFeedID(ctx, feedId)
// 	if err != nil {
// 		return vm, err
// 	}

// 	unreadArticles := []Article{}

// 	for _, v := range unread {
// 		unreadArticles = append(unreadArticles, mapArticleFromUnreadByFeedIDRow(v))
// 	}

// 	vm.IsCache = isCached
// 	vm.PageContent = pageHtmlContent
// 	vm.SidebarData = sbd
// 	vm.PageTitle = afd.ArticleTitle
// 	vm.FeedTitle = afd.FeedTitle
// 	vm.FeedUrl = afd.FeedUrl
// 	vm.Articles = unreadArticles
// 	vm.Link = afd.ArticleLink
// 	vm.ArticleId = afd.ArticleID
// 	vm.FeedId = afd.FeedID
// 	vm.IsStarred = afd.ArticleStarred

// 	return vm, nil

// }

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

func GetFeedUpdates(queries *db.Queries, ctx context.Context) (int64, error) {

	// get all the feed urls, loop through them and pull all the items from the feed
	// for each item in the feed run an insert of ignore statement

	// feeds := []Feed{}
	// err := db.Select(&feeds, "SELECT * from feeds;")
	// if err != nil {
	// 	return 0, fmt.Errorf("error selecting feed data for updates: %v", err)
	// }

	feeds, err := queries.GetFeeds(ctx)
	if err != nil {
		return 0, err
	}

	// godump.Dump(feeds)
	//return 0, nil

	p := gofeed.NewParser()

	//var res sql.Result

	// for _, v := range feeds {
	// 	feed, err := p.ParseURL(v.Url + "/feed/")
	// 	if err != nil {
	// 		return 0, fmt.Errorf("error parsing feed %v", err)
	// 	}
	// 	for _, i := range feed.Items {
	// 		godump.Dump(i)
	// 	}

	// }

	// return 0, errors.New("testing")

	for _, v := range feeds {
		feed, err := p.ParseURL(v.Url + "/feed/")
		if err != nil {
			return 0, fmt.Errorf("error parsing feed %v", err)
		}
		// godump.Dump(feed.Items)
		for _, i := range feed.Items {

			//-----------------------------
			//
			// Will need a more sophisticated strategy here in the long run
			// as we may need to update articles as they can change
			// as we only really use the urls at the moment in the application
			// and dont use the summary of descriptions that are liable to change
			//
			// ----------------------------

			//godump.Dump(i.PublishedParsed)

			pubParsed := ""
			//updatedParsed := ""

			if i.PublishedParsed != nil {
				pubParsed = i.PublishedParsed.Format("2006-01-02 15:04:05")
			}

			// if i.UpdatedParsed != nil {
			// 	updatedParsed = i.UpdatedParsed.Format("2006-01-02 15:04:05")
			// }

			published, err := time.Parse(time.RFC1123Z, i.Published)

			queries.AddToArticles(
				ctx,
				db.AddToArticlesParams{
					FeedID:          v.ID,
					Title:           i.Title,
					Link:            i.Link,
					Published:       published,
					PublishedParsed: pubParsed,
					Summary:         i.Description,
					Read:            0,
					Starred:         0,
				},
			)

			if err != nil {
				return 0, fmt.Errorf("error inserting or replacing articles in feed update: %v", err)
			}
		}
	}

	// rowsChanged, err := res.RowsAffected()
	// if err != nil {
	// 	return 0, fmt.Errorf("error reading affected rows: %v", err)
	// }

	return 0, nil

}

// sqlc mappings to domain
func mapArticleFromLatest5ArticlesRow(row db.GetLatest5ArticlesRow) Article {
	return Article{
		Id:              row.ID,
		FeedId:          row.FeedID,
		Title:           row.Title,
		Link:            row.Link,
		Published:       row.Published.String(),
		PublishedParsed: row.Published.String(),
		Summary:         row.Summary,
		Read:            int64ToBool(row.Read),
		Starred:         int64ToBool(row.Starred),
		FeedTitle:       row.FeedTitle,
	}
}

func mapArticleFromUnreadByFeedIDRow(row db.GetUnreadByFeedIDRow) Article {
	return Article{
		Id:              row.ID,
		FeedId:          row.FeedID,
		Title:           row.Title,
		Link:            row.Link,
		Published:       row.Published.String(),
		PublishedParsed: row.Published.String(),
		Summary:         row.Summary,
		Read:            int64ToBool(row.Read),
		Starred:         int64ToBool(row.Starred),
		FeedTitle:       row.FeedTitle,
	}
}

func mapFeedFromDbFeed(row db.Feed) Feed {
	return Feed{
		Id:                     row.ID,
		Url:                    row.Url,
		Title:                  row.Title,
		LastFetched:            row.LastFetched.String(),
		CSSSelectorContainer:   row.CssSelContainer.String,
		CSSSelectorStart:       row.CssSelStart.String,
		CSSSelectorStop:        row.CssSelStop.String,
		HTMLExtractionStrategy: row.HtmlExtractionStrategy.String,
	}
}

func mapArticleWithFeedDataFromArticleByArticleIDRow(row db.GetFeedDataForArticleByArticleIDRow) ArticleAndFeed {
	return ArticleAndFeed{
		ArticleID:              row.ID,
		ArticleLink:            row.Link,
		ArticleTitle:           row.Title,
		ArticleStarred:         row.Starred,
		FeedID:                 row.FeedID,
		FeedTitle:              row.FeedTitle,
		CssSelContainer:        row.CssSelContainer,
		CssSelStart:            row.CssSelStart,
		CssSelStop:             row.CssSelStop,
		HtmlExtractionStrategy: row.HtmlExtractionStrategy,
	}
}

func mapSidebarLinkFromSidebarDataRow(row db.GetSidebarDataRow) SidebarLink {
	return SidebarLink{
		Name:   row.FeedTitle,
		Link:   fmt.Sprintf("/feed/%v", row.FeedID),
		Unread: (row.TotalArticles - row.ArticlesRead),
	}
}

type Feed struct {
	Id                     int64  `json:"id" db:"id"`
	Url                    string `json:"url" db:"url"`
	Title                  string `json:"title" title:"title"`
	ArticlesRead           int64  `json:"articles_read" db:"articles_read"`
	TotalArticles          int64  `json:"total_articles" db:"total_articles"`
	LastFetched            string `json:"last_fetched" db:"last_fetched"`
	CSSSelectorContainer   string `json:"css_sel_container" db:"css_sel_container"`
	CSSSelectorStart       string `json:"css_sel_start" db:"css_sel_start"`
	CSSSelectorStop        string `json:"css_sel_stop" db:"css_sel_stop"`
	HTMLExtractionStrategy string `json:"html_extraction_strategy" db:"html_extraction_strategy"`
}

func (f Feed) LastFetchedDate() string {
	return "not impl"
}

type Article struct {
	Id              int64  `json:"id" db:"id"`
	FeedId          int64  `json:"feed_id" db:"feed_id"`
	Title           string `json:"title" db:"title"`
	Link            string `json:"link" db:"link"`
	Published       string `json:"published" db:"published"`
	PublishedParsed string `json:"published_parsed" db:"published_parsed"`
	// Updated         string `json:"updated" db:"updated"`
	// UpdatedParsed   string `json:"updated_parsed" db:"updated_parsed"`
	Summary   string `json:"summary" db:"summary"`
	Read      bool   `json:"read" db:"read"`
	Starred   bool   `json:"starred" db:"starred"`
	FeedTitle string `json:"feed_title" db:"feed_title"`
}

type ArticleAndFeed struct {
	ArticleID              int64
	ArticleLink            string
	ArticleTitle           string
	ArticleStarred         int64
	FeedID                 int64
	FeedTitle              string
	FeedUrl                string
	CssSelContainer        sql.NullString
	CssSelStart            sql.NullString
	CssSelStop             sql.NullString
	HtmlExtractionStrategy sql.NullString
}

func (a Article) ScrubbedSummary() template.HTML {
	p := bluemonday.UGCPolicy()
	return template.HTML(p.Sanitize(a.Summary))
}

func (a Article) PublishedDate() string {

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

type SidebarLink struct {
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

type HomepagePageVM struct {
	PageTitle   string
	SidebarData []SidebarLink
	Articles    []Article
}

type FeedPageVM struct {
	FeedId      int64
	PageTitle   string
	SidebarData []SidebarLink
	Articles    []Article
}

type ArticlePageVM struct {
	FeedId      int64
	PageTitle   string
	SidebarData []SidebarLink
	Articles    []Article
	FeedTitle   string
	FeedUrl     string
	Link        string
	PageContent string
	ArticleId   int64
	IsCache     bool
	IsStarred   int64
}

type UpdateParms struct {
	FeedId   int64
	PageType string
}

type feedFormVm struct {
	ButtonText string
	UrlAction  string
	Feed
}
