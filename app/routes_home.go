package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/lib"
	"github.com/starfederation/datastar/sdk/go/datastar"
	"golang.org/x/net/html"
)

func setupHomeRoutes(r chi.Router, queries *db.Queries) {

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feeds, err := queries.SelectAllFeeds(ctx)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		feedSummaries := []FeedSummary{}
		for _, f := range feeds {
			s := FeedSummary{}
			s.Name = f.Title
			s.PageID = 1
			s.FeedID = f.ID
			feedSummaries = append(feedSummaries, s)
		}

		Layout(
			pageProps{Title: "Feeds homepage", Description: ""},
			PageHome(feedSummaries),
		).Render(w)

	})

	r.Get("/feed/{feedID}/page/{pageID}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := requireIDParam(w, r, "feedID")
		if !ok {
			return
		}

		pageID, ok := requireIDParam(w, r, "pageID")
		if !ok {
			return
		}

		feed, err := queries.SelectFeedByID(ctx, feedID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		feeds, err := queries.SelectAllFeeds(ctx)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		feedSummaries := []FeedSummary{}

		for _, f := range feeds {

			fsm := FeedSummary{}
			fsm.Name = f.Title
			fsm.FeedID = f.ID

			// add the complete details where needed
			if f.ID == feed.ID {

				fsm.PageID = pageID
				fsm.ShowArticles = true

				offset := (pageID - 1) * 5

				articles, err := queries.SelectArticlesByFeedIDWithLimit(
					ctx,
					db.SelectArticlesByFeedIDWithLimitParams{
						FeedID: feedID,
						Limit:  5,
						Offset: offset,
					},
				)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				enrichedArticles, err := enrichArticles(queries, ctx, articles)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				fsm.Articles = enrichedArticles

				articleCount, err := queries.SelectArticleCountByFeedID(ctx, fsm.FeedID)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				fsm.ArticleCount = articleCount
				fsm.LinksRequired = int64(math.Ceil(float64(articleCount) / float64(5)))

			}

			feedSummaries = append(feedSummaries, fsm)

		}

		//godump.Dump(feedSummaries)

		var buf bytes.Buffer
		PageHome(feedSummaries).Render(&buf)

		fmt.Println(buf.String())
		sse := datastar.NewSSE(w, r)
		sse.PatchElements(string(buf.String()))

	})

	r.Get("/article/{articleID}/view", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		articleID, ok := requireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		ps, err := getArticlePageData(queries, ctx, articleID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		var buf bytes.Buffer
		PageArticle(ps).Render(&buf)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(buf.String())

	})

	// r.Put("/article/{feedID}/{articleID}/set-read", func(w http.ResponseWriter, r *http.Request) {

	// 	ctx := r.Context()
	// 	feedID, ok := requireIDParam(w, r, "feedID")
	// 	if !ok {
	// 		return
	// 	}
	// 	articleID, ok := requireIDParam(w, r, "articleID")
	// 	if !ok {
	// 		return
	// 	}

	// 	err := queries.UpdateArticleSetAsRead(ctx, articleID)
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	ps, err := getArticlePageData(queries, ctx, articleID, feedID)
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	sse := datastar.NewSSE(w, r)
	// 	sse.PatchElementTempl(TemplateArticlePage(ps))

	// 	/* call an existing JS function  when the new data is morphed in*/
	// 	sse.ExecuteScript("feedsBalanceArticleLayout()")

	// })

	// r.Put("/article/{feedID}/{articleID}/like/{value}", func(w http.ResponseWriter, r *http.Request) {

	// 	ctx := r.Context()

	// 	feedID, ok := requireIDParam(w, r, "feedID")
	// 	if !ok {
	// 		return
	// 	}
	// 	articleID, ok := requireIDParam(w, r, "articleID")
	// 	if !ok {
	// 		return
	// 	}

	// 	likeValue, err := strconv.Atoi(r.PathValue("value"))
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	if likeValue < 0 && likeValue > 3 {
	// 		logAndError(w, r, fmt.Sprintf("incorrect like value: %v, needs to be between 0 and 3", likeValue))
	// 		return
	// 	}

	// 	err = setArticleLike(queries, int64(likeValue), articleID, ctx)
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	ps, err := getArticlePageData(queries, ctx, articleID, feedID)
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	sse := datastar.NewSSE(w, r)
	// 	sse.PatchElementTempl(TemplateArticlePage(ps))
	// 	/* call an existing JS function  when the new data is morphed in*/
	// 	sse.ExecuteScript("feedsBalanceArticleLayout()")
	// })

	r.Get("/article/{articleID}/comment/edit/{paragraphID}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		articleID, ok := requireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		paragraphID, ok := requireNumericParam(w, r, "paragraphID")
		if !ok {
			return
		}

		mns, err := getComments(queries, ctx, articleID, paragraphID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		// We're editing at this point
		mns.ShowTextArea = true

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(TemplateEditComments(mns))
		/* call an existing JS function  when the new data is morphed in*/
		sse.ExecuteScript("feedsBalanceArticleLayout()")

	})

	r.Post("/article/{articleID}/comment/write/{paragraphID}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		articleID, ok := requireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		paragraphID, ok := requireNumericParam(w, r, "paragraphID")
		if !ok {
			return
		}

		// might make sense to error here if empty
		// -------------------------------------------------
		noteText := r.FormValue("note-text")

		mns, err := updateComments(queries, ctx, noteText, articleID, paragraphID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		mns.ShowTextArea = false

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(TemplateEditComments(mns))
		/* call an existing JS function  when the new data is morphed in*/
		sse.ExecuteScript("feedsBalanceArticleLayout()")
	})

	r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

		_, err := GetFeedUpdates(queries, r.Context())
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(
			TemplateRefreshPage(),
			datastar.WithModeAppend(),
			datastar.WithSelector("body"),
		)

	})

}

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
						Val: datastar.GetSSE("/article/%v/paragraph/edit/%v", articleID, paragraphID),
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

type EnrichedArticle struct {
	Article      db.SelectArticlesByFeedIDWithLimitRow
	CommentsData CommentsTemplateData
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

type FeedFormTemplateData struct {
	ButtonText string
	UrlAction  string
	Feed       db.Feed
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
