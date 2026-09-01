package app

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mugtree/feeds/app/db"
)

func setupAdminRoutes(r chi.Router, queries *db.Queries) {

	r.Get("/admin/feeds/list", func(w http.ResponseWriter, r *http.Request) {

		feeds, err := queries.SelectAllFeeds(r.Context())
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}
		TemplateLayout("Feeds list", TemplateAdminListFeeds(feeds)).Render(r.Context(), w)
	})

	r.Get("/admin/feed/{feedID}/view", func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		feedID, ok := requireIDParam(w, r, "feedID")
		if !ok {
			return
		}

		feed, err := queries.SelectFeedByID(ctx, feedID)
		if err != nil {
			logAndError(w, r, err.Error())
			return
		}

		vm := FeedFormTemplateData{Feed: feed, ButtonText: "Update feed"}
		TemplateLayout("Feed view", TemplateAdminFeedForm(vm)).Render(r.Context(), w)
	})

	r.Put("/admin/feed/{feedID}/update", func(w http.ResponseWriter, r *http.Request) {

		feedID, ok := requireIDParam(w, r, "feedID")
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

}
