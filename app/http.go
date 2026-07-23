package app

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mugtree/feeds/app/db"
	"github.com/starfederation/datastar/sdk/go/datastar"
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
	})
	return r
}

func httpFrontEndRoutes(r chi.Router, queries *db.Queries) chi.Router {

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		latest, starred, err := feedsGetHomePageArticleSelections(queries, ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sidebar, err := feedsGetSideBarTemplateData(queries, ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		TemplateLayout(
			"Homepage",
			TemplateHomePage(TemplateNav(sidebar), latest, starred)).Render(ctx,
			w,
		)
	})

	r.Get("/feed/{feedID}/view", func(w http.ResponseWriter, r *http.Request) {

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
		pageTitle := feed.Title

		alreadyRead, toRead, err := feedsGetArticlesByFeedID(queries, feedID, ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sidebar, err := feedsGetSideBarTemplateData(queries, ctx)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		TemplateLayout(
			pageTitle,
			TemplateFeedPage(TemplateNav(sidebar), pageTitle, alreadyRead, toRead)).Render(
			r.Context(),
			w,
		)

	})

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

		td, err := feedsGetArticlePageTemplateData(queries, ctx, articleID, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		TemplateLayout(
			td.PageTitle,
			TemplateArticlePage(td)).Render(
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

		td, err := feedsGetArticlePageTemplateData(queries, ctx, articleID, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(
			TemplateLayout(
				td.PageTitle,
				TemplateArticlePage(td),
			),
		)
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

		td, err := feedsGetArticlePageTemplateData(queries, ctx, articleID, feedID)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		sse := datastar.NewSSE(w, r)
		sse.PatchElementTempl(
			TemplateLayout(
				td.PageTitle,
				TemplateArticlePage(td),
			),
		)

	})

	r.Put("/article/{feedID}/{articleID}/annotate", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		_, ok := httpRequireIDParam(w, r, "feedID")
		if !ok {
			return
		}
		articleID, ok := httpRequireIDParam(w, r, "articleID")
		if !ok {
			return
		}

		err := r.ParseForm()
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		start := r.Form.Get("start")
		end := r.Form.Get("end")

		startPos, err := strconv.Atoi(start)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		endPos, err := strconv.Atoi(end)
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}

		selection := r.Form.Get("selection")
		if selection < "" {
			httpLogAndError(w, r, errors.New("selection param is not set?").Error())
			return
		}

		note := r.Form.Get("note")

		fmt.Println(startPos, endPos, note, articleID, ctx)

	})

	r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

		_, err := feedsGetFeedUpdates(queries, r.Context())
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

func httpAdminRoutes(r chi.Router, queries *db.Queries) chi.Router {

	r.Get("/admin/feeds/list", func(w http.ResponseWriter, r *http.Request) {

		feeds, err := queries.SelectAllFeeds(r.Context())
		if err != nil {
			httpLogAndError(w, r, err.Error())
			return
		}
		TemplateAdminPage(TemplateAdminListFeeds(feeds)).Render(r.Context(), w)
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
		TemplateAdminPage(TemplateAdminFeedForm(vm)).Render(r.Context(), w)
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
		TemplateAdminPage(form).Render(r.Context(), w)
	})

	r.Post("/admin/feed/create", func(w http.ResponseWriter, r *http.Request) {
		form := TemplateAdminFeedForm(FeedFormTemplateData{ButtonText: "Create new"})
		TemplateAdminPage(form).Render(r.Context(), w)
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

func httpRequireIDParam(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return httpRequireNonZeroInt64(chi.URLParam(r, key), key, w, r)
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
