package app

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/go-chi/chi/v5"
	"github.com/goforj/godump"
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

		sse := datastar.NewSSE(w, r)
		sse.PatchElementGostar(PageHome(feedSummaries))

	})

	r.Route("/article/{articleID}", func(r chi.Router) {

		// view article
		r.Get("/view", func(w http.ResponseWriter, r *http.Request) {
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

			sse := datastar.NewSSE(w, r)
			sse.PatchElementGostar(PageArticle(ps))
		})

		r.Get("/comment/{paragraphID}/write", func(w http.ResponseWriter, r *http.Request) {

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
			sse.PatchElementGostar(WriteComments(mns))
			sse.ExecuteScript("feedsBalanceArticleLayout()")

		})

		r.Post("/comment/{paragraphID}/write", func(w http.ResponseWriter, r *http.Request) {

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

			sse := datastar.NewSSE(w, r)
			sse.PatchElementGostar(ViewComments(mns))
			sse.ExecuteScript("feedsBalanceArticleLayout()")
		})

		// Like article
		// ------------------------------------
		r.Put("/like/{value}", func(w http.ResponseWriter, r *http.Request) {

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

			sse := datastar.NewSSE(w, r)
			sse.PatchElementGostar(PageArticle(ps))
			sse.ExecuteScript("feedsBalanceArticleLayout()")
		})

	})

	r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

		_, err := GetFeedUpdates(queries, r.Context())
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementGostar(
			Script(
				Raw(`window.location.reload();`),
			),
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

	r.Route("/admin/feed/{feedID}", func(r chi.Router) {

		r.Get("/view", func(w http.ResponseWriter, r *http.Request) {

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

			godump.Dump("before", feed)

			vm := FeedFormTemplateData{Feed: feed, ButtonText: "Update feed"}
			Layout(
				pageProps{
					Title: "Feed view",
				},
				FeedsAdminForm(vm)).Render(w)
		})

		r.Put("/update", func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			feedID, ok := requireIDParam(w, r, "feedID")
			if !ok {
				return
			}

			sigs := FeedCreateUpdateSignals{}
			err := datastar.ReadSignals(r, &sigs)
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			feed, err := queries.UpdateFeed(ctx, db.UpdateFeedParams{
				Url:             sigs.FeedUrl,
				Title:           sigs.Title,
				CssSelContainer: sigs.CSSSelectorContainer,
				CssSelStart:     sigs.CSSSelectorStart,
				CssSelStop:      sigs.CSSSelectorStop,
				ID:              feedID,
			})
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			godump.Dump("after", feed)

			sse := datastar.NewSSE(w, r)
			sse.PatchElementGostar(
				Script(Raw(`window.location.href = "/admin/feeds"`)),
				datastar.WithSelector("body"), datastar.WithModeAppend(),
			)

		})

		r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
			data := FeedFormTemplateData(FeedFormTemplateData{ButtonText: "Create new"})
			Layout(pageProps{Title: "Create new feed"}, FeedsAdminForm(data)).Render(w)
		})

	})

}
