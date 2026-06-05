package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"errors"
	"fmt"
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
var staticFS embed.FS

func SetupHttpServer(queries *db.Queries, user string, password string) chi.Router {

	r := chi.NewRouter()
	r.Handle("/public/*", neuterDirectoryHandler(http.FileServer(http.FS(staticFS))))
	r.Group(func(site chi.Router) {
		site.Use(basicAuthHandler(user, password))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {

			vm, err := getHomePageData(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			PageTemplate(
				"Homepage",
				SideBarTemplate(vm.SidebarData, r),
				HomePageTemplate(vm)).Render(r.Context(),
				w,
			)
		})

		r.Get("/feed/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := requireIDParam(w, r, "feedID")
			if !ok {
				return
			}

			vm, err := getFeedIndexData(feedID, queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			PageTemplate(
				vm.PageTitle,
				SideBarTemplate(vm.SidebarData, r),
				FeedPageTemplate(vm)).Render(
				r.Context(),
				w,
			)
		})

		r.Route("/article/{feedID}/{articleID}", func(article chi.Router) {

			article.Get("/view", func(w http.ResponseWriter, r *http.Request) {

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}
				articleID, ok := requireIDParam(w, r, "articleID")
				if !ok {
					return
				}

				vm, err := getArticleData(articleID, feedID, queries, r.Context())
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						w.WriteHeader(504)
						w.Write([]byte("taking too long to run service"))

					} else {
						logAndError(w, r, err.Error())
						return
					}
				}

				PageTemplate(
					vm.PageTitle,
					SideBarTemplate(vm.SidebarData, r),
					ArticlePageTemplate(vm)).Render(
					r.Context(),
					w,
				)

			})

			// needs to be a put
			article.Put("/set-read", func(w http.ResponseWriter, r *http.Request) {

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}
				articleID, ok := requireIDParam(w, r, "articleID")
				if !ok {
					return
				}

				err := queries.SetArticleAsRead(r.Context(), articleID)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				vm, err := getArticleData(articleID, feedID, queries, r.Context())
				sse := datastar.NewSSE(w, r)
				sse.PatchElementTempl(
					SideBarTemplate(vm.SidebarData, r),
				)

				sse.PatchElementTempl(
					ToReadTemplate(vm.Articles),
				)
			})

			article.Put("/set-star/{starredValue}", func(w http.ResponseWriter, r *http.Request) {

				ctx := r.Context()

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}
				articleID, ok := requireIDParam(w, r, "articleID")
				if !ok {
					return
				}

				starredValue := r.PathValue("starredValue")
				var flippedValue int64 = 0

				if starredValue != "0" && starredValue != "1" {
					logAndError(w, r, fmt.Sprintf("incorrect starred value: %v, needs to be 0 or 1", starredValue))
					return
				}

				if starredValue == "0" {
					flippedValue = 1
				}

				err := queries.SetArticleStarredValue(ctx,
					db.SetArticleStarredValueParams{
						Starred: flippedValue,
						ID:      articleID},
				)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				sse := datastar.NewSSE(w, r)
				sse.PatchElementTempl(StarredTemplate(feedID, articleID, flippedValue))
			})

		})

		r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

			_, err := GetFeedUpdates(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			sse := datastar.NewSSE(w, r)
			sse.PatchElementTempl(RefreshTemplate(), datastar.WithModeAppend(), datastar.WithSelector("body"))

		})

		// READ ALL - plain old html/text
		r.Get("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {

			dbFeeds, err := queries.GetFeeds(r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			feeds := []Feed{}
			for _, v := range dbFeeds {
				feeds = append(feeds, mapFeedFromDbFeed(v))
			}

			AdminPageTemplate(FeedAdminListTemplate(feeds)).Render(r.Context(), w)
		})

		// NEW FEED - plain old html/text
		r.Get("/admin/feeds/new", func(w http.ResponseWriter, r *http.Request) {
			vm := feedFormVm{}
			vm.ButtonText = "Create new"
			vm.UrlAction = ""
			form := FeedAdminFormTemplate(vm)
			AdminPageTemplate(form).Render(r.Context(), w)
		})

		// CREATE - returns SSE
		r.Post("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {

		})

		// READ - returns text/html
		r.Get("/admin/feeds/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := requireIDParam(w, r, "feedID")
			if !ok {
				return
			}

			f, err := queries.GetFeedByID(r.Context(), feedID)
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			vm := feedFormVm{}
			vm.Feed = mapFeedFromDbFeed(f)
			vm.ButtonText = "Update feed"
			vm.UrlAction = ""

			form := FeedAdminFormTemplate(vm)
			AdminPageTemplate(form).Render(r.Context(), w)
		})

		/**
		  -------
		  These are the SSE actions
		*/

		// UPDATE - returns SSE
		r.Put("/admin/feeds/{feedId}", func(w http.ResponseWriter, r *http.Request) {

			_, ok := requireIDParam(w, r, "feedID")
			if !ok {
				return
			}

		})

		// DELETE
		r.Delete("/admin/feeds/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			_, ok := requireIDParam(w, r, "feedID")
			if !ok {
				return
			}

		})

		type FeedCreateUpdateSignals struct {
			Title                  string `json:"title" title:"title"`
			CSSSelectorContainer   string `json:"css_sel_container" db:"css_sel_container"`
			CSSSelectorStart       string `json:"css_sel_start" db:"css_sel_start"`
			CSSSelectorStop        string `json:"css_sel_stop" db:"css_sel_stop"`
			HTMLExtractionStrategy string `json:"html_extraction_strategy" db:"html_extraction_strategy"`
		}

		r.Post("/admin/feeds/validate", func(w http.ResponseWriter, r *http.Request) {

			fsigs := &FeedCreateUpdateSignals{}

			if err := datastar.ReadSignals(r, fsigs); err != nil {
				logAndError(w, r, errors.New("signals not mapping").Error())
				return
			}

			if fsigs.Title == "" {
				// patch in something here
			}

		})

	})

	return r
}

func neuterDirectoryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func basicAuthHandler(user string, user_password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()

			if ok {
				usernameHash := sha256.Sum256([]byte(username))
				passwordHash := sha256.Sum256([]byte(password))
				expectedUsernameHash := sha256.Sum256([]byte(user))
				expectedPasswordHash := sha256.Sum256([]byte(user_password))

				usernameMatch := subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1
				passwordMatch := subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1

				if usernameMatch && passwordMatch {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
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

func requireIDParam(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return requireNonZeroInt64(chi.URLParam(r, key), key, w, r)
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
