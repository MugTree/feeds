package app

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mugtree/feeds/app/db"
	"github.com/starfederation/datastar/sdk/go/datastar"
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

	// pagination
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

		var buf bytes.Buffer
		PageHome(feedSummaries).Render(&buf)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(string(buf.String()))

	})

	// view article
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

	r.Get("/article/{articleID}/comment/{paragraphID}/write", func(w http.ResponseWriter, r *http.Request) {

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

		var buf bytes.Buffer
		WriteComments(mns).Render(&buf)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(buf.String())
		sse.ExecuteScript("feedsBalanceArticleLayout()")

	})

	r.Post("/article/{articleID}/comment/{paragraphID}/write", func(w http.ResponseWriter, r *http.Request) {

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
		commentText := r.FormValue("comment-text")

		mns, err := updateComments(queries, ctx, commentText, articleID, paragraphID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		mns.ShowTextArea = false

		var buf bytes.Buffer
		ViewComments(mns).Render(&buf)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(buf.String())
		sse.ExecuteScript("feedsBalanceArticleLayout()")
	})

	// Like article
	// ------------------------------------
	r.Put("/article/{articleID}/like/{value}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		articleID, ok := requireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		likeValue, err := strconv.Atoi(r.PathValue("value"))
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		if likeValue < 0 && likeValue > 3 {
			logAndError(w, r, fmt.Sprintf("incorrect like value: %v, needs to be between 0 and 3", likeValue))
			return
		}

		err = setArticleLike(queries, int64(likeValue), articleID, ctx)
		if err != nil {
			logAndError(w, r, err.Error())
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
		sse.ExecuteScript("feedsBalanceArticleLayout()")
	})

	// r.Put("/article/{articleID}/set-read", func(w http.ResponseWriter, r *http.Request) {

	// 	ctx := r.Context()
	// 	articleID, ok := requireIDParam(w, r, "articleID")
	// 	if !ok {
	// 		return
	// 	}

	// 	err := queries.UpdateArticleSetAsRead(ctx, articleID)
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	ps, err := getArticlePageData(queries, ctx, articleID)
	// 	if err != nil {
	// 		logAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	var buf bytes.Buffer

	// 	PageHome()

	// 	sse := datastar.NewSSE(w, r)
	// 		//sse.PatchElementTempl(TemplateArticlePage(ps))

	// 	/* call an existing JS function  when the new data is morphed in*/
	// 	sse.ExecuteScript("feedsBalanceArticleLayout()")

	// })

	r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

		_, err := GetFeedUpdates(queries, r.Context())
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		var buf bytes.Buffer

		RefreshPage().Render(&buf)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(
			buf.String(),
			datastar.WithModeAppend(),
			datastar.WithSelector("body"),
		)

	})

}

func setupAdminRoutes(r chi.Router, queries *db.Queries) {

	r.Get("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {

		feeds, err := queries.SelectAllFeeds(r.Context())
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		pp := pageProps{Title: "Feeds list"}
		Layout(pp, ListFeeds(feeds)).Render(w)
	})

	r.Get("/admin/feed/{feedID}/view", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := requireIDParam(w, r, "feedID")
		if !ok {
			return
		}

		feed, err := queries.SelectFeedByID(ctx, feedID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		vm := FeedFormTemplateData{Feed: feed, ButtonText: "Update feed"}
		Layout(
			pageProps{
				Title: "Feed view",
			},
			FeedsAdminForm(vm)).Render(w)
	})

	r.Put("/admin/feed/{feedID}/update", func(w http.ResponseWriter, r *http.Request) {

		feedID, ok := requireIDParam(w, r, "feedID")
		if !ok {
			return
		}
		fmt.Println(feedID)
	})

	r.Get("/admin/feed/create", func(w http.ResponseWriter, r *http.Request) {
		data := FeedFormTemplateData(FeedFormTemplateData{ButtonText: "Create new"})
		Layout(pageProps{Title: "Create new feed"}, FeedsAdminForm(data)).Render(w)
	})

	// r.Post("/admin/feed/create", func(w http.ResponseWriter, r *http.Request) {
	// 	form := TemplateAdminFeedForm(FeedFormTemplateData{ButtonText: "Create new"})

	// 	TemplateLayout("Create new feed", form).Render(r.Context(), w)
	// })

	type FeedCreateUpdateSignals struct {
		Title                  string `json:"title" title:"title"`
		CSSSelectorContainer   string `json:"css_sel_container" db:"css_sel_container"`
		CSSSelectorStart       string `json:"css_sel_start" db:"css_sel_start"`
		CSSSelectorStop        string `json:"css_sel_stop" db:"css_sel_stop"`
		HTMLExtractionStrategy string `json:"html_extraction_strategy" db:"html_extraction_strategy"`
	}

}
