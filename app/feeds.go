package app

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/jmoiron/sqlx"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
)

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

func sideBarLinks(queries *db.Queries, ctx context.Context) ([]SidebarLink, error) {

	// type SideBarLinkData struct {
	// 	FeedId        int    `json:"feed_id" db:"feed_id"`
	// 	FeedTitle     string `json:"feed_title" db:"feed_title"`
	// 	ArticlesRead  int    `json:"articles_read" db:"articles_read"`
	// 	TotalArticles int    `json:"total_articles" db:"total_articles"`
	// }

	//sld := []SideBarLinkData{}
	// if err := db.SelectContext(ctx, &sld, SqlSideBarMenu); err != nil {
	// 	return items, fmt.Errorf("error getting menu data: %v", err)
	// }

	items := []SidebarLink{}
	data, err := queries.GetSidebarData(ctx)
	if err != nil {
		return items, err
	}

	for _, v := range data {
		items = append(items, SidebarLink{
			Name:   v.FeedTitle,
			Link:   fmt.Sprintf("/feed/%v", v.FeedID),
			Unread: (v.TotalArticles - v.ArticlesRead),
		})
	}

	return items, nil
}

type PageVM struct {
	FeedId      int64
	PageTitle   string
	SidebarMenu []SidebarLink
	Articles    []Article
}

type ArticleVM struct {
	PageVM
	FeedTitle   string
	FeedUrl     string
	Link        string
	PageContent string
	ArticleId   int64
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

// type feedFormErrors = map[string]map[string]string

func homepageVm(_ *sqlx.DB, queries *db.Queries, ctx context.Context) (PageVM, error) {

	vm := PageVM{}

	sbd, err := sideBarLinks(queries, ctx)
	if err != nil {
		return vm, fmt.Errorf("error selecting sidebar data: %v", err)
	}

	latest5Articles, err := queries.GetLatest5Articles(ctx)
	if err != nil {
		return vm, err
	}

	articles := []Article{}

	for _, v := range latest5Articles {
		articles = append(articles, articleFromLatest5ArticlesRow(v))

	}

	vm.Articles = articles
	vm.SidebarMenu = sbd
	vm.PageTitle = "Home"
	vm.FeedId = 0

	return vm, nil
}

var digitCheck = regexp.MustCompile(`^[0-9]+$`)

func validateUpdateParams(pt string, fid string) (UpdateParms, error) {

	u := UpdateParms{}
	if !digitCheck.MatchString(fid) {
		return u, fmt.Errorf("id not numeric %v", 500)
	}

	if pt != PageTypeFeed && pt != PageTypeHome && pt != PageTypeArticle {
		return u, fmt.Errorf("wrong page type%v", pt)
	}

	f, err := strconv.Atoi(fid)
	if err != nil {
		return u, fmt.Errorf("error converting feed id %v", err)
	}

	u.FeedId = int64(f)
	u.PageType = pt
	return u, nil

}

func feedPageVm(feedId int64, _ *sqlx.DB, queries *db.Queries, ctx context.Context) (PageVM, error) {

	vm := PageVM{}

	sidebarData, err := sideBarLinks(queries, ctx)
	if err != nil {
		return vm, fmt.Errorf("error selecting sidebar data:: %v", err)
	}

	feed, err := queries.GetFeedByID(ctx, feedId)
	if err != nil {
		return vm, err
	}

	unreadArticles, err := queries.GetUnreadByFeedID(ctx, feedId)
	if err != nil {
		return vm, err
	}

	articles := []Article{}

	for _, v := range unreadArticles {
		articles = append(articles, articleFromUnreadByFeedIDRow(v))
	}

	vm.Articles = articles
	vm.SidebarMenu = sidebarData
	vm.PageTitle = feed.Title
	vm.FeedId = feed.ID
	return vm, nil
}

func setReadStatusVm(feedId int64, articleId int64, dbx *sqlx.DB, queries *db.Queries, ctx context.Context) (ArticleVM, error) {

	vm := ArticleVM{}

	// if !digitCheck.MatchString(articleId) || !digitCheck.MatchString(feedId) {
	// 	return vm, fmt.Errorf("id not numeric art:%v feed: %v", articleId, feedId)
	// }

	err := queries.SetArticleAsRead(ctx, articleId)
	if err != nil {
		return vm, err
	}

	// result, _ := db.ExecContext(ctx,
	// 	`UPDATE articles SET read = 1 WHERE id = ?;string`, articleId)

	// re, _ := result.RowsAffected()
	// if re == 0 {
	// 	return vm, fmt.Errorf("record %v doesnt exist", articleId)
	// }

	vm, err = articlePageVm(articleId, feedId, dbx, queries, ctx)
	if err != nil {
		return vm, err
	}

	return vm, nil

}

func articlePageVm(articleId int64, feedId int64, _ *sqlx.DB, queries *db.Queries, ctx context.Context) (ArticleVM, error) {

	vm := ArticleVM{}

	// if !digitCheck.MatchString(articleId) || !digitCheck.MatchString(feedId) {
	// 	return vm, fmt.Errorf("id not numeric %c", 500)
	// }

	// type ArticleFeedJoin = struct {
	// 	Id                     int    `json:"id" db:"id"`
	// 	FeedId                 int    `json:"feed_id" db:"feed_id"`
	// 	Link                   string `json:"link" db:"link"`
	// 	CSSSelectorContainer   string `json:"css_sel_container" db:"css_sel_container"`
	// 	CSSSelectorStart       string `json:"css_sel_start" db:"css_sel_start"`
	// 	CSSSelectorStop        string `json:"css_sel_stop" db:"css_sel_stop"`
	// 	Title                  string `json:"title" db:"title"`
	// 	FeedTitle              string `json:"feed_title" db:"feed_title"`
	// 	FeedUrl                string `json:"feed_url" db:"feed_url"`
	// 	HTMLExtractionStrategy string `json:"html_extraction_strategy" db:"html_extraction_strategy"`
	// }

	// ca := ArticleFeedJoin{}

	// err := db.Get(&ca, `
	// 			SELECT
	// 				a.id,
	// 				a.link,
	// 				a.title,
	// 				f.id as feed_id,
	// 				f.title as feed_title,
	// 				f.url as feed_url,
	// 				f.css_sel_container,
	// 				f.css_sel_start,
	// 				f.css_sel_stop,
	// 				f.html_extraction_strategy
	// 			FROM
	// 				articles a
	// 			INNER JOIN feeds f
	// 			ON f.id = a.feed_id where a.id = ?`, articleId)

	// if err != nil {
	// 	return vm, fmt.Errorf("error selecting record %v: %v", articleId, err)
	// }

	ca, err := queries.GetFeedDataForArticleByArticleID(ctx, articleId)
	if err != nil {
		return vm, fmt.Errorf("error selecting record %v: %v", articleId, err)
	}

	var pageHtmlContent = ""

	lc, err := queries.GetCachedByLink(ctx, ca.Link)

	if err == nil {
		pageHtmlContent = lc.ArticleContent.String
	} else {

		if err == sql.ErrNoRows {

			type extractionParams struct {
				Container      string
				ClipStartPoint string
				ClipEndPoint   string
			}

			ep := extractionParams{}
			ep.Container = ca.CssSelContainer.String

			switch ca.HtmlExtractionStrategy.String {
			case "no-clip":
				break
			case "clip-start":
				ep.ClipStartPoint = ca.CssSelStart.String
			case "clip-end":
				ep.ClipEndPoint = ca.CssSelStop.String
			case "clip-between":
				ep.ClipStartPoint = ca.CssSelStart.String
				ep.ClipEndPoint = ca.CssSelStop.String
			}

			//TODO - need to add some timeout values here really
			c := colly.NewCollector()

			c.OnHTML(ep.Container, func(h *colly.HTMLElement) {
				pageHtmlContent = ExtractHTMLRangeFlat(h.DOM, ep.ClipStartPoint, ep.ClipEndPoint)
			})

			if err := c.Visit(ca.Link); err != nil {
				return vm, fmt.Errorf("error using colly to visit page: %v - %v", ca.Link, err)
			}

			insertErr := queries.AddToArticleCache(ctx,
				db.AddToArticleCacheParams{
					Link:           ca.Link,
					ArticleContent: sql.NullString{String: pageHtmlContent},
				},
			)
			// _, insertErr := db.ExecContext(ctx, `
			// 		INSERT INTO article_cache (
			// 			link,
			// 			article_content,
			// 			created
			// 		) VALUES(
			// 			?,
			// 			?,
			// 			CURRENT_TIMESTAMP
			// 		);`,
			// 	ca.Link,
			// 	pageHtmlContent,
			// )

			if insertErr != nil {
				return vm, fmt.Errorf("error adding to article cache: %v", insertErr)
			}
		} else {
			return vm, fmt.Errorf("error querying article cache: %v", err)
		}
	}

	// get other page parts
	// --------------------------------------------------------
	sbd, err := sideBarLinks(queries, ctx)
	if err != nil {
		return vm, fmt.Errorf("error getting side data: %v", err)
	}

	unread, err := queries.GetUnreadByFeedID(ctx, feedId)
	if err != nil {
		return vm, err
	}

	unreadArticles := []Article{}

	for _, v := range unread {
		unreadArticles = append(unreadArticles, articleFromUnreadByFeedIDRow(v))
	}

	// unreadArticles := []Article{}
	// err = db.SelectContext(ctx, &unreadArticles,
	// 	SqlUnreadArticlesByFeed, feedId)
	// if err != nil {
	// 	return vm, fmt.Errorf("error getting feed data: %v", err)
	// }

	vm.PageContent = pageHtmlContent
	vm.SidebarMenu = sbd
	vm.PageTitle = ca.Title
	vm.FeedTitle = ca.FeedTitle
	vm.FeedUrl = ca.FeedUrl
	vm.Articles = unreadArticles
	vm.Link = ca.Link
	vm.ArticleId = ca.ID
	vm.FeedId = ca.FeedID

	return vm, nil

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

func GetFeedUpdates(dbx *sqlx.DB, queries *db.Queries, ctx context.Context) (int64, error) {

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
					Link:            i.Title,
					Published:       published,
					PublishedParsed: pubParsed,
					Summary:         i.Description,
					Read:            0,
					Starred:         0,
				},
			)

			//i.PublishedParsed
			// res, err = dbx.Exec(`
			// 		INSERT OR IGNORE INTO articles (
			// 		feed_id,
			// 		title,
			// 		link,
			// 		published,
			// 		published_parsed,
			// 		updated,
			// 		updated_parsed,
			// 		summary,
			// 		read,
			// 		starred
			// 		) VALUES (
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?,
			// 		 ?
			// 		 );`,
			// 	v.ID,
			// 	i.Title,
			// 	i.Link,
			// 	i.Published,
			// 	pubParsed,
			// 	i.Updated,
			// 	updatedParsed,
			// 	i.Description,
			// 	0,
			// 	0,
			// )

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

const (
	SqlFeedItem = `SELECT * FROM feeds where id = ?;`

	SqlArticlesLatest5 string = `
			SELECT 
				a.*, 
				f.title as feed_title 
			 FROM articles a 
			 INNER JOIN feeds f 
			 ON f.id = a.feed_id 
			 ORDER BY published 
			 DESC LIMIT 0, 5;`

	SqlSideBarMenu string = `
			SELECT
    			f.title AS feed_title,
    			f.id AS feed_id,
    			COUNT(a.id) AS total_articles,
    			COUNT(CASE WHEN a.read <> 0 THEN 1 END) AS articles_read
			FROM feeds f
			LEFT JOIN articles a ON f.id = a.feed_id
			GROUP BY f.id, f.title
			ORDER BY feed_title ASC;`

	SqlArticlesByFeed string = `
			SELECT 
				a.*, 
				f.title as feed_title 
			FROM articles a 
			INNER JOIN feeds f 
			ON f.id = a.feed_id 
			WHERE feed_id = ? 
			ORDER BY a.published_parsed DESC;`

	SqlUnreadArticlesByFeed string = `
			SELECT 
				a.*, 
				f.title as feed_title 
			FROM articles a 
			INNER JOIN feeds f 
			ON f.id = a.feed_id 
			WHERE feed_id = ? AND a.read = 0
			ORDER BY a.published_parsed DESC;`
)

// sqlc mappings to domain
func articleFromLatest5ArticlesRow(row db.GetLatest5ArticlesRow) Article {
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

func articleFromUnreadByFeedIDRow(row db.GetUnreadByFeedIDRow) Article {
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
