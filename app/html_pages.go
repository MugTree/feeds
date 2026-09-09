package app

import (
	"fmt"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type pageProps struct {
	Title       string
	Description string
}

func pageLayout(props pageProps, children ...Node) Node {

	return HTML5(HTML5Props{
		Title:       props.Title,
		Description: props.Description,
		Language:    "en",
		Head: []Node{
			Script(Src("/public/js/datastar.js"), Type("module")),
			Link(Rel("stylesheet"), Href("/public/css/app.css")),
		},
		Body: []Node{Class(""),
			Div(
				ID("pagetop"),
				Header(
					A(
						Href("/"),
						Class("home-link text-3xl font-medium"),
						Text("Home"),
					),
					Text("|"),
					A(
						Href("/admin/feeds"),
						Text("Admin"),
					),
					Div(
						Class("action-area"),
						Button(
							ID("update-button"),
							ds.Indicator("fetching"),
							ds.Attr("disabled", "$fetching"),
							ds.On("click", "@get('/update-reader')"),
							Text("Load"),
						),
						Div(
							Data("show", "$fetching"),
							Style("display: none;"),
							Div(
								Class("spinner"),
								ID("spinner"),
							),
						),
					),
				),
				Main(Group(children)),
			),
			Script(Src("/public/js/feeds.js")),
		},
	})
}

func pageHome(summaries []mpdFeedSummary) Node {
	return Div(
		ID("homepage"),
		Div(
			ID("feeds"),
			partialFeedBoxes(summaries),
		),
		Div(
			ID("article"),
		),
	)
}

func pageArticle(aps mpdArticlePageData) Node {
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
				Li(Text(aps.ArticlePublished)),
				Li(Text(aps.FeedTitle), ds.On("click", "$hideTitle = !$hideTitle")),
				Li(partialLikeArticle(aps.ArticleId, aps.StarValue)),
			),
		),
		Section(
			Class("editor"),
			ID("editor"),
			ds.Signals(
				map[string]any{
					"HasBeenRead":                  aps.ArticleHasBeenRead(),
					"HasScrolledToBottomOfArticle": false},
			),
			Raw(aps.PageContent),
			partialViewComments(aps.CommentsTemplateData),
		),
		partialLikeArticle(aps.ArticleId, aps.StarValue),
		Section(
			H3(ds.On("intersect", "$HasScrolledToBottomOfArticle = true")),
			Button(
				Class("have-read-article"),
				Text("Mark as read"),
				Data("show", "$HasScrolledToBottomOfArticle && !$HasBeenRead"),
				ds.On("click", fmt.Sprintf("/article/%d/%d/set-read", aps.FeedID, aps.ArticleId)),
			),
			P(
				A(Text("Back to top"), Href("#homepage")),
			),
		),
		Script(Src("/public/js/feeds.js")),
	)
}
