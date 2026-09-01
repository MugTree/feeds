package app

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mugtree/feeds/app/db"
)

//go:embed public/css/*.css
//go:embed public/js/*.js
//go:embed public/img/*
var staticFS embed.FS

func SetupHTTPServer(queries *db.Queries, user string, password string) chi.Router {

	r := chi.NewRouter()
	r.Handle("/public/*", neuterDirectory(http.FileServer(http.FS(staticFS))))

	r.Group(func(pages chi.Router) {
		pages.Use(debugHttpRequest)
		setupHomeRoutes(pages, queries)
		setupAdminRoutes(pages, queries)
	})
	return r
}

func neuterDirectory(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func debugHttpRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dumpHttpRequest(r, false, false)
		next.ServeHTTP(w, r)
	})
}

func dumpHttpRequest(r *http.Request, readHeaders bool, readJsonBody bool) {

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

func requireNonZeroInt64(value string, key string, w http.ResponseWriter, r *http.Request) (int64, bool) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		logAndError(w, r, err.Error(), http.StatusBadRequest)
		return 0, false
	}

	if v == 0 {
		logAndError(
			w,
			r,
			fmt.Sprintf("key '%s' must be a non-zero integer", key),
			http.StatusBadRequest,
		)
		return 0, false
	}

	return v, true
}

func requireInt64Param(value string, w http.ResponseWriter, r *http.Request) (int64, bool) {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		logAndError(w, r, err.Error(), http.StatusBadRequest)
		return 0, false
	}

	return v, true
}

func requireIDParam(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return requireNonZeroInt64(chi.URLParam(r, key), key, w, r)
}

func requireNumericParam(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return requireInt64Param(chi.URLParam(r, key), w, r)
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

func logAndError(w http.ResponseWriter, _ *http.Request, msg string, statusCode ...int) {
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
