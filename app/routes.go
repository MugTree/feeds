package app

import (
	"net/http"

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

	// r.Route("/game", func(r chi.Router) {

	// 	r.Get("/", func(w http.ResponseWriter, r *http.Request) {

	// 		articles, err := queries.SelectArticlesByFeedID(r.Context(), 1)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		pageLayout(
	// 			pageProps{
	// 				Title: "Game",
	// 			},
	// 			pageGameHome(articles),
	// 		).Render(w)

	// 	})

	// 	/*

	// 		Both routines need to
	// 		----------------------------

	// 		check url input
	// 		pull the article paragraphs
	// 		split the notes text
	// 		check for a conclusion

	// 		No signals
	// 		--------------
	// 		get any notes based on page id

	// 		With signals
	// 		------------
	// 		read signals
	// 		checks for too many notes and trim the extras

	// 	*/

	// 	r.Get("/article/{articleID}/page/{pageNumber}", func(w http.ResponseWriter, r *http.Request) {

	// 		ctx := r.Context()

	// 		articleID, ok := httpRequireIDParam(w, r, "articleID")
	// 		if !ok {
	// 			return
	// 		}

	// 		pageNumber, ok := httpRequireNumericParam(w, r, "pageNumber")
	// 		if !ok {
	// 			return
	// 		}

	// 		article, err := queries.SelectArticleByID(r.Context(), articleID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		authorParagraphs, lastPage, err := getAuthorParagraphsAsPages(article.ArticleContent, int(pageNumber))
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		pageParagraphs := authorParagraphs[pageNumber-1]

	// 		noteText, err := queries.SelectNotesByArticleIDAndPageID(ctx,
	// 			db.SelectNotesByArticleIDAndPageIDParams{
	// 				PageNumber: pageNumber,
	// 				ArticleID:  articleID,
	// 			})

	// 		if err != nil {
	// 			if err != sql.ErrNoRows {
	// 				httpLogAndError(w, r, err.Error())
	// 				return
	// 			}
	// 		}

	// 		notes := []string{}
	// 		if noteText.NoteText != "" {
	// 			notes = strings.Split(noteText.NoteText, "\n\n")
	// 		}

	// 		pageLayout(
	// 			pageProps{Title: "game"},
	// 			pageGamePlay(
	// 				article,
	// 				authorParagraphs,
	// 				pageParagraphs,
	// 				int(pageNumber),
	// 				notes,
	// 				lastPage,
	// 			)).Render(w)

	// 	})

	// 	type gameSignals = struct {
	// 		NotesText      string `json:"notesText"`
	// 		ParagraphCount int64  `json:"paragraphCount"`
	// 	}

	// 	r.Put("/article/{articleID}/page/{pageNumber}", func(w http.ResponseWriter, r *http.Request) {

	// 		articleID, ok := httpRequireIDParam(w, r, "articleID")
	// 		if !ok {
	// 			return
	// 		}

	// 		pageNumber, ok := httpRequireNumericParam(w, r, "pageNumber")
	// 		if !ok {
	// 			return
	// 		}

	// 		gs := gameSignals{}
	// 		err := datastar.ReadSignals(r, &gs)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		notes := strings.Split(gs.NotesText, "\n\n")
	// 		if len(notes) < int(gs.ParagraphCount) {
	// 			httpLogAndError(w, r, fmt.Errorf("notes count: %v should be longer than %v", len(notes), gs.ParagraphCount).Error())
	// 			return
	// 		}

	// 		ctx := r.Context()
	// 		_, err = queries.UpsertAndReturnNote(
	// 			ctx,
	// 			db.UpsertAndReturnNoteParams{
	// 				ArticleID:  articleID,
	// 				PageNumber: pageNumber,
	// 				NoteText:   gs.NotesText},
	// 		)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		article, err := queries.SelectArticleByID(ctx, articleID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		authorParagraphs, _, err := getAuthorParagraphsAsPages(article.ArticleContent, int(pageNumber))
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		userNotes, err := queries.SelectNotesByArticleID(ctx, articleID)
	// 		if err != nil {
	// 			httpLogAndError(w, r, err.Error())
	// 			return
	// 		}

	// 		godump.Dump(userNotes)

	// 		notesDict := lib.SliceToMap(userNotes, func(n db.Note) int {
	// 			return int(n.ID)
	// 		})

	// 		conclusion := getSolvedParagraphsAsPages(notesDict, authorParagraphs)
	// 		godump.Dump(conclusion[pageNumber-1])
	// 		//godump.Dump(conclusion[pageNumber-1])
	// 		// for i, v := range conclusion[pageNumber] {
	// 		// 	godump.Dump(fmt.Printf("para:%v %v", i, v.Creator))
	// 		// }

	// 		// //godump.Dump("sigs", gs)

	// 		// conclusionNotes := notes[gs.ParagraphCount:]
	// 		// godump.Dump("conclusionNotes", conclusionNotes)

	// 		// retainedParagraphs := authorParagraphs[1:]
	// 		// godump.Dump("retainedParagraphs", retainedParagraphs)

	// 		// newParagraphs := []ArticleParagraph{}
	// 		// for _, c := range conclusionNotes {
	// 		// 	newParagraphs = append(newParagraphs, ArticleParagraph{Text: c})
	// 		// }

	// 		// altered := slices.Insert(retainedParagraphs, 0, newParagraphs)

	// 		// godump.Dump(altered)

	// 		sse := datastar.NewSSE(w, r)

	// 		var i = 0
	// 		sse.PatchElementGostar(Div(ID("conclusion"),
	// 			Map(conclusion, func(pp []ArticleParagraph) Node {
	// 				return Map(pp, func(p ArticleParagraph) Node {
	// 					i++
	// 					return P(Text(p.Text), If(i == 1, Style("font-weight: bold")))
	// 				})
	// 			}),
	// 		),
	// 		)

	// 	})

	// })

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
