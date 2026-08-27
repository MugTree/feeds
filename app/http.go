package app

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/goforj/godump"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/app/scraper"
	"github.com/starfederation/datastar/sdk/go/datastar"
	"golang.org/x/net/html"
)

//go:embed public/css/*.css
//go:embed public/js/*.js
//go:embed public/img/*
var staticFS embed.FS

func HttpSetupServer(queries *db.Queries, user string, password string) chi.Router {

	r := chi.NewRouter()
	r.Handle("/public/*", httpNeuterDirectory(http.FileServer(http.FS(staticFS))))

	r.Group(func(pages chi.Router) {
		pages.Use(httpDebugHttpRequest)
		httpFrontEndRoutes(pages, queries)
		httpAdminRoutes(pages, queries)
		httpApiRoutes(pages, queries)
	})
	return r
}

func httpFrontEndRoutes(r chi.Router, queries *db.Queries) chi.Router {

	r.Get("/home", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		fps := FrontPageSignals{}

		err := datastar.ReadSignals(r, &fps)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		feeds, err := queries.SelectAllFeeds(ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		articlesPerPage := 5
		offset := 0
		pageID := 1
		feedSummaries := []FeedSummary{}

		for _, f := range feeds {

			fsd := FeedSummary{}
			fsd.Name = f.Title
			fsd.PageID = int64(pageID)
			fsd.FeedID = f.ID

			//fsd.ShowArticles = false

			articles, err := queries.SelectArticlesByFeedIDWithLimit(
				ctx,
				db.SelectArticlesByFeedIDWithLimitParams{
					FeedID: f.ID,
					Limit:  int64(articlesPerPage),
					Offset: int64(offset),
				},
			)
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}

			articles, err = feedsEnrichArticles(articles)
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}
			fsd.Articles = articles

			articleCount, err := queries.SelectArticleCountByFeedID(ctx, fsd.FeedID)
			if err != nil {
				httpLogAndError(w, r, err.Error())
				return
			}
			fsd.ArticleCount = articleCount
			fsd.LinksRequired = int64(math.Ceil(float64(articleCount) / float64(5)))
			feedSummaries = append(feedSummaries, fsd)
		}

		sort.Slice(feedSummaries, func(i, j int) bool {
			return feedSummaries[i].Name < feedSummaries[j].Name
		})

		TemplateLayout("new homepage", WIP_TemplateHomePage(feedSummaries, fps)).Render(r.Context(), w)
	})

	// display pagination
	r.Get("/home/feed/{feedID}/page/{pageID}", func(w http.ResponseWriter, r *http.Request) {

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

		fps := FrontPageSignals{}

		err = datastar.ReadSignals(r, &fps)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		fsm := FeedSummary{}
		fsm.FeedID = feed.ID
		fsm.Name = feed.Title
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

		articles, err = feedsEnrichArticles(articles)
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

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(WIP_Inner(fsm, fps))

	})

	// r.Get("/feed/{feedID}/view", func(w http.ResponseWriter, r *http.Request) {

	// 	ctx := r.Context()
	// 	feedID, ok := httpRequireIDParam(w, r, "feedID")
	// 	if !ok {
	// 		return
	// 	}

	// 	feed, err := queries.SelectFeedByID(ctx, feedID)
	// 	if err != nil {
	// 		httpLogAndError(w, r, err.Error())
	// 		return
	// 	}
	// 	pageTitle := feed.Title

	// 	alreadyRead, toRead, err := feedsGetArticlesByFeedID(queries, feedID, ctx)
	// 	if err != nil {
	// 		httpLogAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	sidebar, err := feedsGetSideBarTemplateData(queries, ctx)
	// 	if err != nil {
	// 		httpLogAndError(w, r, err.Error())
	// 		return
	// 	}

	// 	TemplateLayout(
	// 		pageTitle,
	// 		TemplateFeedPage(TemplateNav(sidebar), pageTitle, alreadyRead, toRead)).Render(
	// 		r.Context(),
	// 		w,
	// 	)

	// })

	r.Get("/article/{feedID}/{articleID}/view", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}
		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		ps, err := feedsGetArticlePageTemplateData(queries, ctx, articleID, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		TemplateLayout(
			ps.PageTitle,
			TemplateArticlePage(ps)).Render(
			r.Context(),
			w,
		)
	})

	r.Put("/article/{feedID}/{articleID}/set-read", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}
		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		err := queries.UpdateArticleSetAsRead(ctx, articleID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		ps, err := feedsGetArticlePageTemplateData(queries, ctx, articleID, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(TemplateArticlePage(ps))

		/* call an existing JS function  when the new data is morphed in*/
		sse.ExecuteScript("feedsBalanceArticleLayout()")

	})

	r.Put("/article/{feedID}/{articleID}/like/{value}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}
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

		err = feedsSetArticleLike(queries, int64(likeValue), articleID, ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		ps, err := feedsGetArticlePageTemplateData(queries, ctx, articleID, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(TemplateArticlePage(ps))
		/* call an existing JS function  when the new data is morphed in*/
		sse.ExecuteScript("feedsBalanceArticleLayout()")
	})

	r.Get("/article/{articleID}/note/edit/{blockID}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		blockID, ok := httpRequireNumericParam(w, r, "blockID")
		if !ok {
			return
		}

		mns, err := feedsGetComments(queries, ctx, articleID, blockID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		// We're editing at this point
		mns.ShowTextArea = true

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(TemplateEditComments(mns))
		/* call an existing JS function  when the new data is morphed in*/
		sse.ExecuteScript("feedsBalanceArticleLayout()")

	})

	r.Post("/article/{articleID}/note/write/{blockID}", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		blockID, ok := httpRequireNumericParam(w, r, "blockID")
		if !ok {
			return
		}

		// might make sense to error here if empty
		// -------------------------------------------------
		noteText := r.FormValue("note-text")

		mns, err := feedsUpdateComments(queries, ctx, noteText, articleID, blockID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		mns.ShowTextArea = false

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(TemplateEditComments(mns))
		/* call an existing JS function  when the new data is morphed in*/
		sse.ExecuteScript("feedsBalanceArticleLayout()")
	})

	r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

		_, err := scraper.GetFeedUpdates(queries, r.Context())
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(
			TemplateRefreshPage(),
			datastar.WithModeAppend(),
			datastar.WithSelector("body"),
		)

	})

	return r
}

// api can run locally on feeds.localhost or something like that

// Can this will be where the routes that are accessed by the browser plugin live
func httpApiRoutes(r chi.Router, _ *db.Queries) chi.Router {

	r.Post("/api/html/add", func(w http.ResponseWriter, r *http.Request) {

		htmlInput := "" //r.FormValue("")

		rd := strings.NewReader(htmlInput)
		tree, err := html.Parse(rd)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		godump.Dump(tree)

		// echo the html back to the other side of the page

	})

	// needs to display all the fields to allow a user to add a feed
	r.Post("/api/feed/add", func(w http.ResponseWriter, r *http.Request) {

	})

	// alter an existing feed from the browser
	r.Get("/api/feed/alter", func(w http.ResponseWriter, r *http.Request) {})

	return r
}

func httpAdminRoutes(r chi.Router, queries *db.Queries) chi.Router {

	r.Get("/admin/feeds/list", func(w http.ResponseWriter, r *http.Request) {

		feeds, err := queries.SelectAllFeeds(r.Context())
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}
		TemplateLayout("Feeds list", TemplateAdminListFeeds(feeds)).Render(r.Context(), w)
	})

	r.Get("/admin/feed/{feedID}/view", func(w http.ResponseWriter, r *http.Request) {

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

		vm := FeedFormTemplateData{Feed: feed, ButtonText: "Update feed"}
		TemplateLayout("Feed view", TemplateAdminFeedForm(vm)).Render(r.Context(), w)
	})

	r.Put("/admin/feed/{feedID}/update", func(w http.ResponseWriter, r *http.Request) {

		feedID, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}
		fmt.Println(feedID)
	})

	r.Get("/admin/feed/create", func(w http.ResponseWriter, r *http.Request) {
		form := TemplateAdminFeedForm(FeedFormTemplateData{ButtonText: "Create new"})
		TemplateLayout("Create new feed", form).Render(r.Context(), w)
	})

	r.Post("/admin/feed/create", func(w http.ResponseWriter, r *http.Request) {
		form := TemplateAdminFeedForm(FeedFormTemplateData{ButtonText: "Create new"})

		TemplateLayout("Create new feed", form).Render(r.Context(), w)
	})

	type FeedCreateUpdateSignals struct {
		Title                  string `json:"title" title:"title"`
		CSSSelectorContainer   string `json:"css_sel_container" db:"css_sel_container"`
		CSSSelectorStart       string `json:"css_sel_start" db:"css_sel_start"`
		CSSSelectorStop        string `json:"css_sel_stop" db:"css_sel_stop"`
		HTMLExtractionStrategy string `json:"html_extraction_strategy" db:"html_extraction_strategy"`
	}

	return r
}

func httpNeuterDirectory(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// func httpBasicAuthHandler(user string, user_password string) func(http.Handler) http.Handler {
// 	return func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			username, password, ok := r.BasicAuth()

// 			if ok {
// 				usernameHash := sha256.Sum256([]byte(username))
// 				passwordHash := sha256.Sum256([]byte(password))
// 				expectedUsernameHash := sha256.Sum256([]byte(user))
// 				expectedPasswordHash := sha256.Sum256([]byte(user_password))

// 				usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
// 				passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1

// 				if usernameMatch && passwordMatch {
// 					next.ServeHTTP(w, r)
// 					return
// 				}
// 			}

// 			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		})
// 	}
// }

func httpDebugHttpRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpDumpRequest(r, false, false)
		next.ServeHTTP(w, r)
	})
}

func httpDumpRequest(r *http.Request, readHeaders bool, readJsonBody bool) {

	fmt.Printf("\n=== %s %s ===\n", r.Method, r.URL)

	routeCtx := chi.RouteContext(r.Context())
	if routeCtx != nil {
		fmt.Println("Path params:")
		for i, key := range routeCtx.URLParams.Keys {
			fmt.Printf("  %s = %s\n", key, routeCtx.URLParams.Values[i])
		}
	}

	fmt.Println("Query params:")
	for key, values := range r.URL.Query() {
		fmt.Printf("  %s = %v\n", key, values)
	}

	if err := r.ParseForm(); err == nil {
		fmt.Println("Form values:")
		for key, values := range r.PostForm {
			fmt.Printf("  %s = %v\n", key, values)
		}
	}

	if readHeaders {
		fmt.Println("Headers:")
		for key, values := range r.Header {
			fmt.Printf("  %s = %v\n", key, values)
		}
	}

	if readJsonBody {
		fmt.Println("JSON body:")
		body, _ := io.ReadAll(r.Body)
		fmt.Println(string(body))
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}

}

func httpRequireNonZeroInt64(value string, key string, w http.ResponseWriter, r *http.Request) (int64, bool) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		httpLogAndError(w, r, err.Error(), http.StatusBadRequest)
		return 0, false
	}

	if v == 0 {
		httpLogAndError(
			w,
			r,
			fmt.Sprintf("key '%s' must be a non-zero integer", key),
			http.StatusBadRequest,
		)
		return 0, false
	}

	return v, true
}

func httpRequireInt64Param(value string, w http.ResponseWriter, r *http.Request) (int64, bool) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		httpLogAndError(w, r, err.Error(), http.StatusBadRequest)
		return 0, false
	}

	return v, true
}

func httpRequireIDParam(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return httpRequireNonZeroInt64(chi.URLParam(r, key), key, w, r)
}

func httpRequireNumericParam(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return httpRequireInt64Param(chi.URLParam(r, key), w, r)
}

// func requirePageType(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
// 	pt := r.PathValue(key)

// 	switch pt {
// 	case PageTypeFeed, PageTypeHome, PageTypeArticle:
// 		return pt, true
// 	default:
// 		logAndError(w, r, fmt.Errorf("invalid page type: %s", pt).Error())
// 		return "", false
// 	}
// }

func httpLogAndError(w http.ResponseWriter, _ *http.Request, msg string, statusCode ...int) {
	status := 500
	if len(statusCode) > 0 {
		status = statusCode[0]
	}

	_, file, line, ok := runtime.Caller(1) // 1 = caller of this function
	if ok {
		msg = fmt.Sprintf("%s (at %s:%d)", msg, file, line)
	}
	LogError(msg)
	w.WriteHeader(status)
	// ErrorPageTemplate().Render(r.Context(), w)
}
