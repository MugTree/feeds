package app

import (
	"fmt"
	"html/template"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/lib"
)

type EnrichedArticle struct {
	Article      db.SelectArticlesByFeedIDWithLimitRow
	CommentsData CommentsTemplateData
}

type feedsArticle struct {
	Id        int64  `json:"id" db:"id"`
	FeedId    int64  `json:"feed_id" db:"feed_id"`
	Title     string `json:"title" db:"title"`
	Link      string `json:"link" db:"link"`
	Published string `json:"published" db:"published"`
	DateFound string `json:"date_found" db:"date_found"`
	Summary   string `json:"summary" db:"summary"`
	Read      bool   `json:"read" db:"read"`
	Liked     int64  `json:"starred" db:"starred"`
	FeedTitle string `json:"feed_title" db:"feed_title"`
}

func (a feedsArticle) FullName() string {
	return a.FeedTitle + " - " + a.Title
}

func (a feedsArticle) ScrubbedSummary() template.HTML {
	p := bluemonday.UGCPolicy()
	return template.HTML(p.Sanitize(a.Summary))
}

func (a feedsArticle) PublishedDate() string {

	d, err := time.Parse(time.RFC1123Z, a.Published)
	if err != nil {
		fmt.Printf("time parse issue: %v", err)
		return ""
	}

	day := d.Day()
	month := d.Format("January")
	year := d.Year()

	suffix := "th"
	if day%10 == 1 && day != 11 {
		suffix = "st"
	} else if day%10 == 2 && day != 12 {
		suffix = "nd"
	} else if day%10 == 3 && day != 13 {
		suffix = "rd"
	}

	return fmt.Sprintf("%d%s %s %d", day, suffix, month, year)
}

type feedsSidebarLink struct {
	Name   string
	Link   string
	Unread int64
	FeedId int
}

type ArticlePageTemplateData struct {
	FeedID                  int64
	PageTitle               string
	ArticlesRead            []feedsArticle
	ArticlesToRead          []feedsArticle
	FeedTitle               string
	FeedUrl                 string
	Link                    string
	PageContent             string
	ArticleId               int64
	IsCache                 bool
	StarValue               int64
	Sidebar                 []feedsSidebarLink
	ArticlePublished        string
	ArticleRead             int64
	MarginNotes             map[int64]db.Comment
	ClickableParagraphCount int64
	CommentsTemplateData    CommentsTemplateData
}

func (ae ArticlePageTemplateData) ArticleHasBeenRead() bool {
	return lib.IntToBool(ae.ArticleRead)
}

type FeedFormTemplateData struct {
	ButtonText string
	UrlAction  string
	Feed       db.Feed
}

type ArticleStatus struct {
	HasBeenRead                  bool
	HasScrolledToBottomOfArticle bool
}

type CommentsTemplateData struct {
	ShowTextArea                bool
	ArticleID                   int64
	NoteToEdit                  int64
	TotalPotentialCommentsCount int64
	Comments                    map[int64]db.Comment
}

type FeedSummary struct {
	Name          string
	ArticleCount  int64
	FeedID        int64
	PageID        int64
	LinksRequired int64
	Articles      []EnrichedArticle //[]db.SelectArticlesByFeedIDWithLimitRow
	ShowArticles  bool
}

type NewHomePageTemplateData struct {
	FeedMeta       []FeedSummary
	ArticlesByFeed map[string][]db.SelectArticlesByFeedIDWithLimitRow
}

const layoutISO = "2006-01-02"

type FrontPageSignals struct {
	ArticlesOpen []int64 `json:"articlesOpen"`
	FeedsOpen    []int64 `json:"feedsOpen"`
}

type PageScrapeParams struct {
	Link           string
	Container      string
	ClipStartPoint string
	ClipEndPoint   string
	Strategy       string
}

type FeedCreateUpdateSignals struct {
	Title                  string `json:"feed-name"`
	FeedUrl                string `json:"feed-url"`
	CSSSelectorContainer   string `json:"css-sel-container"`
	CSSSelectorStart       string `json:"css-sel-start"`
	CSSSelectorStop        string `json:"css-sel-stop"`
	HTMLExtractionStrategy string `json:"html-extraction-strategy"`
}
