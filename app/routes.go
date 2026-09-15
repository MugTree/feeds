package app

import (
	"net/http"
	"strings"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/go-chi/chi/v5"
	"github.com/goforj/godump"
	"github.com/mugtree/feeds/app/db"
	"github.com/starfederation/datastar/sdk/go/datastar"
)

func routesHomePage(r chi.Router, queries *db.Queries) {

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feeds, err := queries.SelectAllFeeds(ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		feedSummaries := []mpdFeedSummary{}
		for _, f := range feeds {
			s := mpdFeedSummary{}
			s.Name = f.Title
			s.PageID = 1
			s.FeedID = f.ID
			feedSummaries = append(feedSummaries, s)
		}

		pageLayout(
			pageProps{
				Title:       "Feeds homepage",
				Description: "",
			},
			pageHome(feedSummaries),
		).Render(w)

	})

	r.Route("/game", func(r chi.Router) {

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {

			articles, err := queries.SelectArticlesByFeedID(r.Context(), 1)
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}

			pageLayout(
				pageProps{
					Title: "Game",
				},
				pageGameHome(articles),
			).Render(w)

		})

		r.Get("/article/{articleID}/page/{pageNumber}", func(w http.ResponseWriter, r *http.Request) {

			articleID, ok := httpRequireIDParam(w, r, "articleID")
			if !ok {
				return
			}

			pageNumber, ok := httpRequireNumericParam(w, r, "pageNumber")
			if !ok {
				return
			}

			article, err := queries.SelectArticleByID(r.Context(), articleID)
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}

			paragraphs, err := getArticleParagraphsByPageNumber(article.ArticleContent, int(pageNumber))
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}

			pageLayout(
				pageProps{
					Title: article.Published.String(),
				},
				pageGamePlay(article, paragraphs, int(pageNumber)),
			).Render(w)

		})

		type noteSignals = struct {
			Text           string `json:"text"`
			ParagraphCount int64  `json:"paragraphCount"`
		}

		r.Put("/article/{articleID}/page/{pageNumber}/notes/add", func(w http.ResponseWriter, r *http.Request) {

			articleID, ok := httpRequireIDParam(w, r, "articleID")
			if !ok {
				return
			}

			pageNumber, ok := httpRequireNumericParam(w, r, "pageNumber")
			if !ok {
				return
			}

			godump.Dump("articleID", articleID, "pageNumber", pageNumber)

			ns := noteSignals{}
			err := datastar.ReadSignals(r, &ns)
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}

			ns.Text = strings.Trim(ns.Text, "\n")
			notes := strings.Split(ns.Text, "\n\n")

			godump.Dump("signals", &ns)
			godump.Dump("para count ", ns.ParagraphCount)

			// are there more that +1 more notes than paragraphs
			// if so remove them an just carry on
			if len(notes) > int(ns.ParagraphCount)+1 {
				notes = notes[0 : int(ns.ParagraphCount)+1]
			}

			godump.Dump("notes", notes)
			/*




			 */
			sse := datastar.NewSSE(w, r)
			sse.PatchElementGostar(Div(ID("notes"), Map(notes, func(n string) Node {
				return P(Text(n))
			})))

			sse.ExecuteScript(`equaliseHeights();`)

		})

	})

	// r.Get("/feed/{feedID}/page/{pageID}", func(w http.ResponseWriter, r *http.Request) {

	// 	ctx := r.Context()

	// 	feedID, ok := httpRequireIDParam(w, r, "feedID")
	// 	if !ok {
	// 		return
	// 	}

	// 	pageID, ok := httpRequireIDParam(w, r, "pageID")
	// 	if !ok {
	// 		return
	// 	}

	// 	feed, err := queries.SelectFeedByID(ctx, feedID)
	// 	if err != nil {
	// 		httpLogAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	feeds, err := queries.SelectAllFeeds(ctx)
	// 	if err != nil {
	// 		httpLogAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	feedSummaries := []mpdFeedSummary{}

	// 	for _, f := range feeds {

	// 		fsm := mpdFeedSummary{}
	// 		fsm.Name = f.Title
	// 		fsm.FeedID = f.ID

	// 		// add the complete details where needed
	// 		if f.ID == feed.ID {

	// 			fsm.PageID = pageID
	// 			fsm.ShowArticles = true

	// 			offset := (pageID - 1) * 5

	// 			articles, err := queries.SelectArticlesByFeedIDWithLimit(
	// 				ctx,
	// 				db.SelectArticlesByFeedIDWithLimitParams{
	// 					FeedID: feedID,
	// 					Limit:  5,
	// 					Offset: offset,
	// 				},
	// 			)
	// 			if err != nil {
	// 				httpLogAndError(w, r, err.Error())
	// 				return
	// 			}

	// 			enrichedArticles, err := enrichArticles(queries, ctx, articles)
	// 			if err != nil {
	// 				httpLogAndError(w, r, err.Error())
	// 				return
	// 			}

	// 			fsm.Articles = enrichedArticles

	// 			articleCount, err := queries.SelectArticleCountByFeedID(ctx, fsm.FeedID)
	// 			if err != nil {
	// 				httpLogAndError(w, r, err.Error())
	// 				return
	// 			}

	// 			fsm.ArticleCount = articleCount
	// 			fsm.LinksRequired = int64(math.Ceil(float64(articleCount) / float64(5)))

	// 		}

	// 		feedSummaries = append(feedSummaries, fsm)

	// 	}

	// 	sse := datastar.NewSSE(w, r)
	// 	sse.PatchElementGostar(pageHome(feedSummaries))

	// })

	// r.Route("/article/{articleID}", func(r chi.Router) {

	// 	// view article
	// 	r.Get("/view", func(w http.ResponseWriter, r *http.Request) {
	// 		ctx := r.Context()

	// 		articleID, ok := httpRequireIDParam(w, r, "articleID")
	// 		if !ok {
	// 			return
	// 		}

	// 		articleData, err := getArticlePageData(queries, ctx, articleID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		sse := datastar.NewSSE(w, r)
	// 		sse.PatchElementGostar(pageArticle(articleData))
	// 	})

	// 	r.Get("/comment/{paragraphID}/write", func(w http.ResponseWriter, r *http.Request) {

	// 		ctx := r.Context()

	// 		articleID, ok := httpRequireIDParam(w, r, "articleID")
	// 		if !ok {
	// 			return
	// 		}

	// 		paragraphID, ok := httpRequireNumericParam(w, r, "paragraphID")
	// 		if !ok {
	// 			return
	// 		}

	// 		commentsData, err := getComments(queries, ctx, articleID, paragraphID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		// We're editing at this point
	// 		commentsData.ShowTextArea = true

	// 		sse := datastar.NewSSE(w, r)
	// 		sse.PatchElementGostar(partialWriteComments(commentsData))
	// 		sse.ExecuteScript("feedsBalanceArticleLayout()")

	// 	})

	// 	r.Post("/comment/{paragraphID}/write", func(w http.ResponseWriter, r *http.Request) {

	// 		ctx := r.Context()

	// 		articleID, ok := httpRequireIDParam(w, r, "articleID")
	// 		if !ok {
	// 			return
	// 		}

	// 		paragraphID, ok := httpRequireNumericParam(w, r, "paragraphID")
	// 		if !ok {
	// 			return
	// 		}

	// 		// might make sense to error here if empty
	// 		// -------------------------------------------------
	// 		commentText := r.FormValue("comment-text")

	// 		mns, err := updateComments(queries, ctx, commentText, articleID, paragraphID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		mns.ShowTextArea = false

	// 		sse := datastar.NewSSE(w, r)
	// 		sse.PatchElementGostar(partialViewComments(mns))
	// 		sse.ExecuteScript("feedsBalanceArticleLayout()")
	// 	})

	// 	// Like article
	// 	// ------------------------------------
	// 	r.Put("/like/{value}", func(w http.ResponseWriter, r *http.Request) {

	// 		ctx := r.Context()

	// 		articleID, ok := httpRequireIDParam(w, r, "articleID")
	// 		if !ok {
	// 			return
	// 		}

	// 		likeValue, err := strconv.Atoi(r.PathValue("value"))
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		if likeValue < 0 && likeValue > 3 {
	// 			httpLogAndError(w, r, fmt.Sprintf("incorrect like value: %v, needs to be between 0 and 3", likeValue))
	// 			return
	// 		}

	// 		err = setArticleLike(queries, int64(likeValue), articleID, ctx)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		ps, err := getArticlePageData(queries, ctx, articleID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		sse := datastar.NewSSE(w, r)
	// 		sse.PatchElementGostar(pageArticle(ps))
	// 		sse.ExecuteScript("feedsBalanceArticleLayout()")
	// 	})

	// })

	r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

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

	})

}

func routesAdminPage(r chi.Router, queries *db.Queries) {

	r.Get("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {

		feeds, err := queries.SelectAllFeeds(r.Context())
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		pp := pageProps{Title: "Feeds list"}
		pageLayout(pp, partialListFeeds(feeds)).Render(w)
	})

	r.Route("/admin/feed/{feedID}", func(r chi.Router) {

		r.Get("/view", func(w http.ResponseWriter, r *http.Request) {

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
			pageLayout(
				pageProps{
					Title: "Feed view",
				},
				partialFeedsAdminForm(vm)).Render(w)
		})

		r.Put("/update", func(w http.ResponseWriter, r *http.Request) {

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

		})

		r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
			data := FeedFormTemplateData(FeedFormTemplateData{ButtonText: "Create new"})
			pageLayout(pageProps{Title: "Create new feed"}, partialFeedsAdminForm(data)).Render(w)
		})

	})

}
