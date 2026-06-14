package app

import (
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"slices"
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

func SetupHttpServer(queries *db.Queries, user string, password string) chi.Router {

	r := chi.NewRouter()
	r.Handle("/public/*", neuterDirectoryHandler(http.FileServer(http.FS(staticFS))))
	r.Group(func(site chi.Router) {
		site.Use(basicAuthHandler(user, password))

		/**
		-------------------------------------
		Simple homepage
		-------------------------------------
		*/
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			latest, starred, err := getHomepageArticles(queries, ctx)
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			sidebar, err := getSidebarData(queries, ctx)
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			PageTemplate(
				"Homepage",
				HomePageTemplate(NavTemplate(sidebar), latest, starred)).Render(ctx,
				w,
			)
		})

		r.Route("/feed/{feedID}", func(feed chi.Router) {

			/**
			------------------------------------------------
			Feed index page - show all the articles per feed
			------------------------------------------------
			*/
			feed.Get("/view", func(w http.ResponseWriter, r *http.Request) {

				ctx := r.Context()

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}

				feed, err := getFeed(queries, feedID, ctx)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}
				pageTitle := feed.Title

				alreadyRead, toRead, err := getArticlesByFeed(queries, feedID, ctx)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				sidebar, err := getSidebarData(queries, ctx)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				PageTemplate(
					pageTitle,
					FeedPageTemplate(NavTemplate(sidebar), pageTitle, alreadyRead, toRead)).Render(
					r.Context(),
					w,
				)
			})

		})

		r.Route("/article/{feedID}/{articleID}", func(article chi.Router) {

			/**
			---------------------------------------------------------------------
			Main page - read an article that's been picked up bu the feed reader
			---------------------------------------------------------------------
			*/
			article.Get("/view", func(w http.ResponseWriter, r *http.Request) {

				ctx := r.Context()

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}
				articleID, ok := requireIDParam(w, r, "articleID")
				if !ok {
					return
				}

				td, err := getArticleTemplateData(queries, ctx, articleID, feedID)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				PageTemplate(
					td.PageTitle,
					ArticlePageTemplate(td)).Render(
					r.Context(),
					w,
				)
			})

			/**
			-----------------------------------------------
			Called at the bottom of an article on intersect
			-----------------------------------------------
			*/
			article.Put("/set-read", func(w http.ResponseWriter, r *http.Request) {

				ctx := r.Context()

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}
				articleID, ok := requireIDParam(w, r, "articleID")
				if !ok {
					return
				}

				err := queries.SetArticleAsRead(ctx, articleID)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				td, err := getArticleTemplateData(queries, ctx, articleID, feedID)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				sse := datastar.NewSSE(w, r)
				sse.PatchElementTempl(
					PageTemplate(
						td.PageTitle,
						ArticlePageTemplate(td),
					),
				)

			})

			/**
			------------------------------------------------------------
			Set on a user click to set a star value on the article pages
			------------------------------------------------------------
			*/
			article.Put("/like/{value}", func(w http.ResponseWriter, r *http.Request) {

				ctx := r.Context()

				feedID, ok := requireIDParam(w, r, "feedID")
				if !ok {
					return
				}
				articleID, ok := requireIDParam(w, r, "articleID")
				if !ok {
					return
				}

				likeValue, err := strconv.Atoi(r.PathValue("value"))
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				if !slices.Contains([]int{0, 1, 2, 3}, likeValue) {
					logAndError(w, r, fmt.Sprintf("incorrect like value: %v, needs to be between 0 and 3", likeValue))
					return
				}

				err = setArticleLikeValue(queries, int64(likeValue), articleID, ctx)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				td, err := getArticleTemplateData(queries, ctx, articleID, feedID)
				if err != nil {
					logAndError(w, r, err.Error())
					return
				}

				sse := datastar.NewSSE(w, r)
				sse.PatchElementTempl(
					PageTemplate(
						td.PageTitle,
						ArticlePageTemplate(td),
					),
				)

			})

		})

		/**
		-------------------------------------------------------
		At the moment this reloads all of the feeds and returns
		a JS window.location.reload();
		--------------------------------------------------------
		*/
		r.Get("/update-reader", func(w http.ResponseWriter, r *http.Request) {

			_, err := getFeedUpdates(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			sse := datastar.NewSSE(w, r)
			sse.PatchElementTempl(
				RefreshTemplate(),
				datastar.WithModeAppend(),
				datastar.WithSelector("body"),
			)

		})

		// READ ALL - plain old html/text
		r.Get("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {
			feeds, err := getAllFeeds(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			AdminPageTemplate(FeedAdminListTemplate(feeds)).Render(r.Context(), w)
		})

		// NEW FEED - plain old html/text
		r.Get("/admin/feeds/new", func(w http.ResponseWriter, r *http.Request) {
			form := FeedAdminFormTemplate(FeedFormTemplateData{ButtonText: "Create new"})
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

			ctx := r.Context()

			feed, err := getFeed(queries, feedID, ctx)
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			vm := FeedFormTemplateData{Feed: feed, ButtonText: "Update feed"}
			AdminPageTemplate(FeedAdminFormTemplate(vm)).Render(r.Context(), w)
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
