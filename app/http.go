package app

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/goforj/godump"
	"github.com/mugtree/feeds/app/db"
	"github.com/starfederation/datastar/sdk/go/datastar"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

//go:embed public/css/*.css
//go:embed public/js/*.js
//go:embed public/img/*
var staticFS embed.FS

func SetupHTTPServer(queries *db.Queries, user string, password string) chi.Router {

	r := chi.NewRouter()
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

		feedSummaries := []mpdFeedSummary{}
		for _, f := range feeds {
			s := mpdFeedSummary{}
			s.Name = f.Title
			s.PageID = 1
			s.FeedID = f.ID
			feedSummaries = append(feedSummaries, s)
		}

		TemplateLayout(
			pageProps{
				Title:       "Feeds homepage",
				Description: "",
			},
			TemplateHomePage(feedSummaries),
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

		sse := datastar.NewSSE(w, r)
		sse.PatchElementGostar(TemplateArticlePage(articleData))
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

		feedSummaries := []mpdFeedSummary{}

		for _, f := range feeds {

			fsm := mpdFeedSummary{}
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

		pp := pageProps{Title: "Feeds list"}
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
			pageProps{
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

	r.Get("/", homePageHandler)
	r.Get("/feed/{feedID}/page/{pageID}", feedPageHandler)
	r.Get("/article/{articleID}/view", articlePageHandler)
	r.Get("/article/{articleID}/write-comment", writeCommentHandler)
	r.Put("/article/{articleID}/like/{value}", likeArticleHander)
	r.Get("/update-reader", updateReaderHandler)
	r.Get("/admin/feeds", listFeedsAdminHandler)
	r.Get("/admin/feed/{feedID}/view", viewFeedAdminHandler)
	r.Put("/admin/feed/{feedID}/update", updateFeedAdminHandler)

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

func httpDebugRequest(next http.Handler) http.Handler {
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

// func feedsGetSideBarTemplateData(queries *db.Queries, ctx context.Context) ([]feedsSidebarLink, error) {

// 	items := []feedsSidebarLink{}
// 	data, err := queries.SelectSideBarData(ctx)
// 	if err != nil {
// 		return items, err
// 	}

// 	for _, row := range data {
// 		items = append(items, feedsSidebarLink{
// 			Name:   row.FeedTitle,
// 			Link:   fmt.Sprintf("/feed/%v/view", row.FeedID),
// 			Unread: (row.TotalArticles - row.ArticlesRead),
// 		})
// 	}

// 	return items, nil
// }
