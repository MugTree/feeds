package app

import (
	"fmt"
	"strconv"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

func getFeedUrl(feedID int64) string {
	return fmt.Sprintf("@get('/feed/%v/page/1')", feedID)
}

func PageHome(summaries []FeedSummary) Node {
	return Div(
		ID("homepage"),
		Div(
			ID("feeds"),
			Map(summaries, func(fs FeedSummary) Node {
				return FeedBox(getFeedUrl(fs.FeedID), fs)
			},
			),
		),
		Div(
			ID("article"),
		),
	)
}

func FeedBox(link string, fsm FeedSummary) Node {

	pagination := func(fsm FeedSummary) Node {

		pageNums := []int64{}
		for i := range fsm.LinksRequired {
			pageNums = append(pageNums, i+1)
		}

		return Map(pageNums, func(i int64) Node {
			return A(
				ds.On("click", fmt.Sprintf("@get('/feed/%v/page/%v')", fsm.FeedID, i)),
				Text(strconv.FormatInt(i, 10)),
				Classes{"link": true, "underline": fsm.PageID == i},
			)
		},
		)
	}

	return Div(
		ID(fmt.Sprintf("feed-%v", fsm.FeedID)),
		H2(Text(fsm.Name), ds.On("click", link)),
		If(len(fsm.Articles) > 0,
			Div(
				Map(fsm.Articles,
					func(a EnrichedArticle) Node {
						articleURL := fmt.Sprintf("@get('/article/%v/view')", a.Article.ArticleID)
						return H3(Text(a.Article.ArticleTitle), ds.On("click", articleURL))
					},
				),
				Div(pagination(fsm)),
			),
		),
	)
}

func PageArticle(aps ArticlePageTemplateData) Node {
	return Div(
		ID("article"),
		ds.Signals(map[string]any{"hideTitle": false}),
		Div(
			ID("article-top"),
			H2(
				Text(fmt.Sprintf("%v - %v", aps.FeedTitle, aps.PageTitle)),
				Data("show", "!$hideTitle"),
			),
			Ul(
				Li(B(Text(aps.FeedTitle))),
				Li(Text(aps.ArticlePublished)),
				Li(Text(aps.FeedTitle), ds.On("click", "$hideTitle = !$hideTitle")),
			),
		),
		Section(
			Class("editor"),
			ds.Signals(
				map[string]any{
					"HasBeenRead":                  aps.ArticleHasBeenRead(),
					"HasScrolledToBottomOfArticle": false},
			),
			Raw(aps.PageContent),
			ViewComments(aps.CommentsTemplateData),
		),
		LikeUrl("star-value-bottom", aps.FeedID, aps.ArticleId, aps.StarValue),
		Section(
			H3(ds.On("intersect", "$HasScrolledToBottomOfArticle = true")),
			Button(
				Text("Mark as read"),
				Data("show", "$HasScrolledToBottomOfArticle && !$HasBeenRead"),
				ds.On("click", fmt.Sprintf("/article/%d/%d/set-read", aps.FeedID, aps.ArticleId)),
			),
			P(
				A(Text("Back to top"), Href("#pagetop")),
			),
		),
		Script(Src("/public/js/feeds.js")),
	)
}

func LikeUrl(htmlID string, feedID int64, articleID int64, starsValue int64) Node {
	return Div(
		ID(htmlID),
		ds.On("click", fmt.Sprintf("@put('/article/%v/%v/like/%v')", feedID, articleID, starsValue)),
		Text("Starred:"),
		Img(
			Width("60px"),
			Src(fmt.Sprintf("/public/img/%v-star.png", starsValue)),
		),
	)
}

func ViewComments(mns CommentsTemplateData) Node {
	slots := []int64{}
	for i := range mns.TotalPotentialCommentsCount {
		slots = append(slots, i)
	}
	return Aside(
		ID("article-notes"),
		Map(slots, func(i int64) Node {
			note, _ := mns.Comments[i]
			return Div(
				Data("note-id", strconv.FormatInt(i, 10)),
				Class("note-holder"),
				P(Text(note.CommentText)),
			)
		}),
	)
}
