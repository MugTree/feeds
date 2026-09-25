package app

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/goforj/godump"
	"github.com/mugtree/feeds/app/db"
	"github.com/starfederation/datastar/sdk/go/datastar"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func setupRoutes(r chi.Router, queries *db.Queries) chi.Router {

	r.Use(httpDebugRequest)
	r.Handle("/public/*", httpNeuterDirectory(http.FileServer(http.FS(staticFS))))

	/*
		--------------------------------------------
		Front end functionality
		--------------------------------------------
	*/
	homePageHandler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		feeds, err := queries.SelectAllFeeds(ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		summaries := []FeedSummary{}
		for _, f := range feeds {
			s := FeedSummary{}
			s.Name = f.Title
			s.PageID = 1
			s.FeedID = f.ID
			summaries = append(summaries, s)
		}

		TemplateLayout(
			TemplatePageProps{
				Title:       "Feeds homepage",
				Description: "",
			},
			TemplateHomePage(summaries),
		).Render(w)
	}

	articlePageHandler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		articleData, err := getArticlePageData(queries, ctx, articleID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		TemplateLayout(
			TemplatePageProps{
				Title: articleData.PageTitle,
			},
			TemplateArticlePage(articleData),
		).Render(w)

	}

	feedPageHandler := func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}

		pageID, ok := httpRequireIDParam(w, r, "pageID")
		if !ok {
			return
		}

		feed, err := queries.SelectFeedByID(ctx, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		feeds, err := queries.SelectAllFeeds(ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
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
					httpLogAndError(w, r, err.Error())
					return
				}

				fsm.Articles = articles

				articleCount, err := queries.SelectArticleCountByFeedID(ctx, fsm.FeedID)
				if err != nil {
					httpLogAndError(w, r, err.Error())
					return
				}

				fsm.ArticleCount = articleCount
				fsm.LinksRequired = int64(math.Ceil(float64(articleCount) / float64(5)))

			}

			feedSummaries = append(feedSummaries, fsm)

		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementGostar(TemplateHomePage(feedSummaries))

	}

	writeCommentHandler := func(w http.ResponseWriter, r *http.Request) {

		// This will take a position on page param and just tack it back to page using css
	}

	likeArticleHander := func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		likeValue, err := strconv.Atoi(r.PathValue("value"))
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		if likeValue < 0 && likeValue > 3 {
			httpLogAndError(w, r, fmt.Sprintf("incorrect like value: %v, needs to be between 0 and 3", likeValue))
			return
		}

		err = setArticleLike(queries, int64(likeValue), articleID, ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		ps, err := getArticlePageData(queries, ctx, articleID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementGostar(TemplateArticlePage(ps))
		sse.ExecuteScript("feedsBalanceArticleLayout()")
	}

	updateReaderHandler := func(w http.ResponseWriter, r *http.Request) {

		_, err := getFeedUpdates(queries, r.Context())
		if err != nil {
			httpLogAndError(w, r, err.Error())
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

	}

	r.Get("/", homePageHandler)
	r.Get("/feed/{feedID}/page/{pageID}", feedPageHandler)
	r.Get("/article/{articleID}/view", articlePageHandler)
	r.Get("/article/{articleID}/write-comment", writeCommentHandler)
	r.Put("/article/{articleID}/like/{value}", likeArticleHander)
	r.Get("/update-reader", updateReaderHandler)

	/*
		----------------------------------------------
		Admin functionality
		----------------------------------------------
	*/

	listFeedsAdminHandler := func(w http.ResponseWriter, r *http.Request) {
		feeds, err := queries.SelectAllFeeds(r.Context())
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		pp := TemplatePageProps{Title: "Feeds list"}
		TemplateLayout(pp, partialListFeeds(feeds)).Render(w)
	}

	viewFeedAdminHandler := func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}

		feed, err := queries.SelectFeedByID(ctx, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		godump.Dump("before", feed)

		vm := FeedFormTemplateData{Feed: feed, ButtonText: "Update feed"}
		TemplateLayout(
			TemplatePageProps{
				Title: "Feed view",
			},
			TemplateFeedsPageAdminForm(vm)).Render(w)
	}

	updateFeedAdminHandler := func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}

		sigs := mpdCreateFeedSignals{}
		err := datastar.ReadSignals(r, &sigs)
		if err != nil {
			httpLogAndError(w, r, err.Error())
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
			httpLogAndError(w, r, err.Error())
			return
		}

		godump.Dump("after", feed)

		sse := datastar.NewSSE(w, r)
		sse.PatchElementGostar(
			Script(Raw(`window.location.href = "/admin/feeds"`)),
			datastar.WithSelector("body"), datastar.WithModeAppend(),
		)

	}

	r.Get("/admin/feeds", listFeedsAdminHandler)
	r.Get("/admin/feed/{feedID}/view", viewFeedAdminHandler)
	r.Put("/admin/feed/{feedID}/update", updateFeedAdminHandler)

	return r

}
