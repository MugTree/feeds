package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/lib"
	"github.com/starfederation/datastar/sdk/go/datastar"
	"golang.org/x/net/html"
)

func feedsGetArticlePageTemplateData(queries *db.Queries, ctx context.Context, articleID int64, feedID int64) (ArticlePageTemplateData, error) {

	td := ArticlePageTemplateData{}

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

	td.ClickableBlockCount = fa.ArticleClickableBlockCount

	if fa.ArticleContent != "" {
		td.PageContent = fa.ArticleContent
	} else {
		td.PageContent = "<html><head></head><body><p>some dummy content</p></body>"
	}

	enrichedHTML, err := feedsEnrichHTMLOutput(td.PageContent, feedID, articleID)
	if err != nil {
		return td, err
	}

	td.PageContent = enrichedHTML
	td.IsCache = true

	mns, err := feedsGetMarginNotes(queries, ctx, articleID, -1)

	td.MarginNotesTemplateData = mns

	return td, nil

	// scrapedHTML, err := feedsScrapeSiteHTML(queries, fa, ctx)
	// if err != nil {
	// 	if errors.Is(err, context.DeadlineExceeded) {
	// 		return td, err
	// 	}
	// 	return td, err
	// }

	// processedHTML, clickableBlocksCount, err := feedsProcessScrapedHTML(scrapedHTML)
	// if err != nil {
	// 	return td, err
	// }

	// newlyCached, err := queries.InsertAndReturnCachedArticle(ctx, db.InsertAndReturnCachedArticleParams{
	// 	ArticleID:           articleID,
	// 	Link:                td.Link,
	// 	ArticleContent:      sql.NullString{String: processedHTML, Valid: true},
	// 	ClickableBlockCount: clickableBlocksCount,
	// })

	// if err != nil {
	// 	return td, err
	// }

	// enrichedHTMLForUser, err := feedsEnrichHTMLOutput(newlyCached.ArticleContent.String, feedID, articleID)

	// td.PageContent = enrichedHTMLForUser
	// td.ClickableBlockCount = newlyCached.ClickableBlockCount

	// return td, nil
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

/* This needs to update or insert a specific margin note and then return all the margin notes */
func feedsUpdateMarginNotes(queries *db.Queries, ctx context.Context, noteText string, articleID int64, blockID int64) (MarginNotesTemplateData, error) {

	mns := MarginNotesTemplateData{}

	fmt.Printf("Does a note already exist - block id: %v - note:%s\n", blockID, noteText)

	_, err := queries.SelectMarginNoteByArticleIDAndBlockID(
		ctx, db.SelectMarginNoteByArticleIDAndBlockIDParams{
			ArticleID: articleID,
			BlockID:   blockID,
		},
	)

	//  If a note doesn't exist to update we INSERT a new one
	if err == sql.ErrNoRows {
		fmt.Println("No!")
		fmt.Printf("Creating a new note - block id: %v - note:%s and returning all the notes\n", blockID, noteText)

		_, err := queries.InsertAndReturnMarginNote(
			ctx,
			db.InsertAndReturnMarginNoteParams{
				Note:      noteText,
				ArticleID: articleID,
				BlockID:   blockID,
			},
		)
		if err != nil {
			return mns, err
		}

		return feedsGetMarginNotes(queries, ctx, articleID, blockID)
	}

	if err != nil {
		return mns, err
	}

	fmt.Println("Yes!")
	fmt.Printf("Updating an existing note - block id: %v - note:%s and returning all the notes\n", blockID, noteText)

	err = queries.UpdateMarginNoteByArticleIDAndBlockID(
		ctx,
		db.UpdateMarginNoteByArticleIDAndBlockIDParams{
			Note:      noteText,
			ArticleID: articleID,
			BlockID:   blockID,
		},
	)
	if err != nil {
		return mns, err
	}

	return feedsGetMarginNotes(queries, ctx, articleID, blockID)

}

func feedsGetMarginNotes(queries *db.Queries, ctx context.Context, articleID int64, blockID int64) (MarginNotesTemplateData, error) {

	mns := MarginNotesTemplateData{}

	// CLARIFY!!!! if this is -1 then its the page render call
	fmt.Printf("Selecting note state: %v\n", blockID)
	mns.NoteToEdit = blockID

	article, err := queries.SelectArticleByID(ctx, articleID)
	if err != nil {
		return mns, err
	}
	mns.TotalBlocksCount = article.ClickableBlockCount
	mns.ArticleID = article.ID

	notes, err := queries.SelectMarginNotesByArticleID(ctx, articleID)
	if err != nil {
		return mns, err
	}

	getBlockID := func(n db.MarginNote) int64 {
		return n.BlockID
	}

	notesMap := lib.SliceToMap(notes, getBlockID)
	mns.MarginNotes = notesMap

	return mns, nil
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

// func feedsGetHomePageArticleSelections(queries *db.Queries, ctx context.Context) (latest []feedsArticle, starred []feedsArticle, err error) {

// 	latest5Articles, err := queries.SelectLatest5Articles(ctx)
// 	if err != nil {
// 		return latest, starred, err
// 	}

// 	for _, row := range latest5Articles {
// 		latest = append(latest, feedsArticle{
// 			Id:        row.ID,
// 			FeedId:    row.FeedID,
// 			Title:     row.Title,
// 			Link:      row.Link,
// 			Published: row.Published.Format(layoutISO),
// 			DateFound: row.DateFound.Format(layoutISO),
// 			Summary:   row.Summary,
// 			Read:      lib.IntToBool(row.Read),
// 			Liked:     row.Starred,
// 			FeedTitle: row.FeedTitle,
// 		})
// 	}

// 	starredArticles, err := queries.SelectLatest5StarredArticles(ctx)

// 	for _, row := range starredArticles {
// 		starred = append(starred, feedsArticle{
// 			Id:        row.ID,
// 			FeedId:    row.FeedID,
// 			Title:     row.Title,
// 			Link:      row.Link,
// 			Published: row.Published.Format(layoutISO),
// 			DateFound: row.Published.Format(layoutISO),
// 			Summary:   row.Summary,
// 			Read:      lib.IntToBool(row.Read),
// 			Liked:     row.Starred,
// 			FeedTitle: row.FeedTitle,
// 		})
// 	}

// 	fmt.Printf("starred: %v", len(starred))

// 	return latest, starred, err

// }

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

// func feedsGetArticleContentIfCached(queries *db.Queries, articleLink string, _ int64, ctx context.Context) (bool, string, int64, error) {

// 	var clickableBlocks int64 = 0

// 	lc, err := queries.SelectCachedArticleByLink(ctx, articleLink)
// 	if err == nil {

// 		// parsedHTML, err := html.Parse(strings.NewReader(lc.ArticleContent.String))
// 		// if err != nil {
// 		// 	return false, "", 0, err
// 		// }

// 		// article, err := feedsRemoveOuterHTMLShell(parsedHTML)
// 		// if err != nil {
// 		// 	return false, "", clickableBlocks, err
// 		// }

// 		// articleStr, err := feedsStringifyHTML(article)
// 		// if err != nil {
// 		// 	return false, "", clickableBlocks, err
// 		// }

// 		clickableBlocks := lc.ClickableBlockCount

// 		return true, lc.ArticleContent.String, clickableBlocks, nil
// 	}
// 	if err == sql.ErrNoRows {
// 		return false, "", clickableBlocks, nil
// 	}

// 	return false, "", clickableBlocks, err

// }

/* returns the fully processed information plus some data about the processing */

func feedsEnrichArticles(articles []db.SelectArticlesByFeedIDWithLimitRow) ([]db.SelectArticlesByFeedIDWithLimitRow, error) {

	for i := range articles {
		if articles[i].ArticleContent != "" {
			enrichedContent, err := feedsEnrichHTMLOutput(
				articles[i].ArticleContent,
				0,
				articles[i].ArticleID,
			)

			if err != nil {
				return articles, err
			}

			articles[i].ArticleContent = enrichedContent
		}
	}

	return articles, nil

}

/* before data is passed to the front end we add some additional properties for interactivity*/
func feedsEnrichHTMLOutput(htmlStr string, _ int64, articleID int64) (string, error) {

	addDataAttributes := func(doc *html.Node) *html.Node {

		var walk func(*html.Node)

		count := 0

		walk = func(n *html.Node) {

			count++

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
						Val: datastar.GetSSE("/article/%v/note/edit/%v", articleID, blockID),
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

	transformed, err := feedsStringifyHTML(htmlNodes)
	if err != nil {
		return "", err
	}

	return transformed, nil

}

func feedsStringifyHTML(doc *html.Node) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

/* adding some properties to the HTML coming that we are ingesting */

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
	MarginNotes             map[int64]db.MarginNote
	ClickableBlockCount     int64
	MarginNotesTemplateData MarginNotesTemplateData
}

func (ae ArticlePageTemplateData) ArticleHasBeenRead() bool {
	return lib.IntToBool(ae.ArticleRead)
}

type FeedFormTemplateData struct {
	ButtonText string
	UrlAction  string
	Feed       db.Feed
}

type ArticleStatus struct {
	HasBeenRead                  bool
	HasScrolledToBottomOfArticle bool
}

type MarginNotesTemplateData struct {
	ShowTextArea     bool
	ArticleID        int64
	NoteToEdit       int64
	TotalBlocksCount int64
	MarginNotes      map[int64]db.MarginNote
}

type FeedSummary struct {
	Name          string
	ArticleCount  int64
	FeedID        int64
	PageID        int64
	LinksRequired int64
	Articles      []db.SelectArticlesByFeedIDWithLimitRow
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
