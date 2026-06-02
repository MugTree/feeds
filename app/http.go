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
	"github.com/jmoiron/sqlx"
	"github.com/mugtree/feeds/app/db"
	"github.com/starfederation/datastar/sdk/go/datastar"
)

//go:embed public/css/*.css
//go:embed public/js/*.js
var staticFS embed.FS

func SetupHttpServer(dbx *sqlx.DB, queries *db.Queries, user string, password string) chi.Router {

	r := chi.NewRouter()
	r.Handle("/public/*", neuterDirectoryHandler(http.FileServer(http.FS(staticFS))))
	r.Group(func(site chi.Router) {
		site.Use(basicAuthHandler(user, password))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {

			homeVm, err := getHomepageData(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			PageTemplate(
				"Homepage",
				SideBarTemplate(homeVm.SidebarMenu, r),
				HomePageMainTemplate(homeVm)).Render(r.Context(),
				w,
			)
		})

		r.Get("/feed/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}

			feedVm, err := getFeedPageData(feedID, queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			PageTemplate(
				feedVm.PageTitle,
				SideBarTemplate(feedVm.SidebarMenu, r),
				FeedPageMainTemplate(feedVm)).Render(
				r.Context(),
				w,
			)
		})

		r.Get("/article/{feedID}/{articleID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}
			articleID, ok := paramMustBeNonZeroNumeric(w, r, "articleID")
			if !ok {
				return
			}

			articleVm, err := getArticlePageData(articleID, feedID, queries, r.Context())
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
				articleVm.PageTitle,
				SideBarTemplate(articleVm.SidebarMenu, r),
				ArticlePageMainTemplate(articleVm)).Render(
				r.Context(),
				w,
			)
		})

		r.Get("/set-as-read/{feedID}/{articleID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}
			articleID, ok := paramMustBeNonZeroNumeric(w, r, "articleID")
			if !ok {
				return
			}

			readStatusVm, err := setReadStatusVm(feedID, articleID, queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			sse := datastar.NewSSE(w, r)
			sse.PatchElementTempl(
				SideBarTemplate(readStatusVm.SidebarMenu, r),
			)

			sse.PatchElementTempl(
				ToReadTemplate(readStatusVm.Articles),
			)
		})

		r.Get("/update/{pageType}/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}

			pageType, ok := pageTypeMustBeInRange(w, r, "pageType")
			if !ok {
				return
			}

			sse := datastar.NewSSE(w, r)
			sse.PatchElementTempl(UpdatingFeedButtonTemplate(pageType, feedID))
		})

		r.Get("/updating/{pageType}/{feedId}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}

			pageType, ok := pageTypeMustBeInRange(w, r, "pageType")
			if !ok {
				return
			}

			_, err := GetFeedUpdates(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			type pageParts struct {
				SidebarMenu []SidebarLink
				Articles    []Article
			}

			pp := pageParts{}

			sbl, err := getSideBarLinks(queries, r.Context())
			if err != nil {
				logAndError(w, r, err.Error())
				return
			}

			pp.SidebarMenu = sbl

			switch pageType {
			case PageTypeHome:

				// latest5Articles := []Article{}
				// if err := dbx.SelectContext(r.Context(), &latest5Articles,
				// 	SqlArticlesLatest5,
				// ); err != nil {
				// 	logAndError(w, r, fmt.Errorf("error getting latest 5 articles: %v", err).Error())
				// 	return
				// }

				latest5Articles, err := queries.GetLatest5Articles(r.Context())
				if err != nil {
					logAndError(w, r, fmt.Errorf("error getting latest 5 articles: %v", err).Error())
					return
				}

				articles := []Article{}

				for _, v := range latest5Articles {
					articles = append(articles, mapArticleFromLatest5ArticlesRow(v))
				}

				pp.Articles = articles

			case PageTypeFeed, PageTypeArticle:

				// feedArticlesById := []Article{}
				// err = dbx.SelectContext(r.Context(), &feedArticlesById,
				// 	SqlUnreadArticlesByFeed, vp.FeedId)
				// if err != nil {
				// 	logAndError(w, r, fmt.Errorf("error getting unread articles by feed: %v", err).Error())
				// 	return
				// }

				feedArticlesByID, err := queries.GetUnreadByFeedID(r.Context(), feedID)
				if err != nil {
					logAndError(w, r, fmt.Errorf("error getting uread articles: %v", err).Error())
					return
				}

				articles := []Article{}

				for _, v := range feedArticlesByID {
					articles = append(articles, mapArticleFromUnreadByFeedIDRow(v))
				}

				pp.Articles = articles

			default:
				logAndError(w, r, fmt.Errorf("incorrect page type: %v", pageType).Error())
			}

			sse := datastar.NewSSE(w, r)
			sse.PatchElementTempl(SideBarTemplate(pp.SidebarMenu, r))
			sse.PatchElementTempl(ToReadTemplate(pp.Articles))
			sse.PatchElementTempl(UpdateFeedButtonTemplate(pageType, feedID))

		})

		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<!DOCTYPE html><html><head><title>health</title></head><body></body></html>"))
		})

		// READ ALL - plain old html/text
		r.Get("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {
			f := []Feed{}
			dbx.Select(&f, "SELECT * FROM feeds;")
			AdminPageMainTemplate(FeedAdminListTemplate(f)).Render(r.Context(), w)
		})

		// NEW FEED - plain old html/text
		r.Get("/admin/feeds/new", func(w http.ResponseWriter, r *http.Request) {
			vm := feedFormVm{}
			vm.ButtonText = "Create new"
			vm.UrlAction = ""
			form := FeedAdminFormTemplate(vm)
			AdminPageMainTemplate(form).Render(r.Context(), w)
		})

		// CREATE - returns SSE
		r.Post("/admin/feeds", func(w http.ResponseWriter, r *http.Request) {

		})

		// READ - returns text/html
		r.Get("/admin/feeds/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			feedID, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}

			f, err := queries.GetFeedByID(r.Context(), feedID)
			if err != nil {
				logAndError(w, r, err.Error())
			}

			vm := feedFormVm{}
			vm.Feed = mapFeedFromDbFeed(f)
			vm.ButtonText = "Update feed"
			vm.UrlAction = ""

			form := FeedAdminFormTemplate(vm)
			AdminPageMainTemplate(form).Render(r.Context(), w)
		})

		/**
		  -------
		  These are the SSE actions
		*/

		// UPDATE - returns SSE
		r.Put("/admin/feeds/{feedId}", func(w http.ResponseWriter, r *http.Request) {

			_, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
			if !ok {
				return
			}

		})

		// DELETE
		r.Delete("/admin/feeds/{feedID}", func(w http.ResponseWriter, r *http.Request) {

			_, ok := paramMustBeNonZeroNumeric(w, r, "feedID")
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

func mustBeNonZeroNumeric(w http.ResponseWriter, r *http.Request, key, value string) (int64, bool) {
	v, err := strconv.Atoi(value)
	if err != nil {
		logAndError(w, r, err.Error(), http.StatusBadRequest)
		return 0, false
	}

	v64 := int64(v)

	if v64 == 0 {
		logAndError(
			w,
			r,
			fmt.Sprintf("key '%v' must be non-zero numeric", key),
			http.StatusBadRequest,
		)
		return 0, false
	}

	return v64, true
}

func paramMustBeNonZeroNumeric(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	return mustBeNonZeroNumeric(w, r, key, chi.URLParam(r, key))
}

func pageTypeMustBeInRange(w http.ResponseWriter, r *http.Request, key string) (string, bool) {

	pt := r.PathValue(key)
	if pt != PageTypeFeed && pt != PageTypeHome && pt != PageTypeArticle {
		logAndError(w, r, fmt.Errorf("wrong page type%v", pt).Error())
		return "", false
	}

	return pt, true

}

// func postMustBeNonZeroNumeric(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
// 	return mustBeNonZeroNumeric(w, r, key, r.PostFormValue(key))
// }

// func paramMustBeNotEmpty(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
// 	v := chi.URLParam(r, key)
// 	if v == "" {
// 		logAndError(w, r, fmt.Errorf("key '%v' empty string - %v", key, v).Error())
// 		return "", false
// 	}
// 	return v, true
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
