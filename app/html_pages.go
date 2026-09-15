package app

import (
	"fmt"

	"github.com/mugtree/feeds/app/db"
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
			//Link(Rel("stylesheet"), Href("/public/css/app.css")),
			Link(Rel("stylesheet"), Href("/public/css/main.css")),
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

func pageGameHome(articles []db.SelectArticlesByFeedIDRow) Node {
	return Ul(
		Map(articles, func(a db.SelectArticlesByFeedIDRow) Node {
			return Li(
				A(
					Text(a.Published.String()),
					Href(fmt.Sprintf("/game/article/%v/page/1", a.ID)),
				),
			)
		}),
	)
}

func pageGamePlay(article db.Article, articleParagraphs [][]articleParagraph, pageNumber int) Node {

	showNextLink := func(pageNumber int) bool {
		totalParagraphs := 0
		for _, paras := range articleParagraphs {
			totalParagraphs += len(paras)
		}
		return totalParagraphs > pageNumber*PARAGRAPHS_PER_PAGE
	}

	showPreviousLink := func(pageNumber int) bool {
		return pageNumber > 1
	}

	pageIndex := pageNumber - 1
	pageParagraphs := articleParagraphs[pageIndex]

	return Div(Class("grid-parent"),

		Div(Class("paragraphs"),
			ds.Signals(map[string]any{"paragraphCount": len(pageParagraphs)}),
			Map(pageParagraphs, func(p articleParagraph) Node {
				return P(
					Text(p.Text),
				)
			}),
			Div(
				Textarea(
					ds.Bind("text"),
					ds.On("input", `twoConsecutiveNewlines(evt) && @put("/game/article/notes")`),
					ID("user_input"),
				),
			),
			Div(
				If(showPreviousLink(pageNumber),
					A(
						Text("< Previous chunk"),
						Href(fmt.Sprintf("/game/article/%v/page/%v", article.ID, pageNumber-1)),
					)),
				Span(Text("|")),
				If(showNextLink(pageNumber),
					A(
						Text("Next chunk >"),
						Href(fmt.Sprintf("/game/article/%v/page/%v", article.ID, pageNumber+1)),
					),
				),
			),
		),
		Div(ID("notes")),
		Div(Class("overview"),

			Map(articleParagraphs, func(paras []articleParagraph) Node {
				return Map(paras, func(p articleParagraph) Node {
					return P(Text(p.Text))
				})
			}),

			//Raw(article.ArticleContent),
			// Map(paragraphs, func(para []articleParagraph) Node {

			// 	cla := Classes{}

			// 	if mapIter == pageNumber {
			// 		cla = Classes{"current-paragraphs": true}
			// 	} else {
			// 		cla = Classes{"other-paragraphs": true}
			// 	}

			// 	mapIter++

			// 	return Map(para, func(p articleParagraph) Node {
			// 		return P(Text(p.Text), cla)
			// 	})
			// }),
		),
		Script(Src("/public/js/feeds.js")),
	)
}

func pageArticle(apd mpdArticlePageData) Node {
	return Div(
		ID("article"),
		ds.Signals(map[string]any{"hideTitle": false}),
		Div(
			ID("article-top"),
			H2(
				Text(fmt.Sprintf("%v - %v", apd.FeedTitle, apd.PageTitle)),
				Data("show", "!$hideTitle"),
			),
			Ul(
				Li(Text(apd.ArticlePublished)),
				Li(Text(apd.FeedTitle), ds.On("click", "$hideTitle = !$hideTitle")),
				Li(partialLikeArticle(apd.ArticleId, apd.StarValue)),
			),
		),
		Section(
			Class("editor"),
			ID("editor"),
			ds.Signals(
				map[string]any{
					"HasBeenRead":                  apd.ArticleHasBeenRead(),
					"HasScrolledToBottomOfArticle": false},
			),
			Raw(apd.PageContent),
			partialViewComments(apd.CommentsTemplateData),
		),
		partialLikeArticle(apd.ArticleId, apd.StarValue),
		Section(
			H3(ds.On("intersect", "$HasScrolledToBottomOfArticle = true")),
			Button(
				Class("have-read-article"),
				Text("Mark as read"),
				Data("show", "$HasScrolledToBottomOfArticle && !$HasBeenRead"),
				ds.On("click", fmt.Sprintf("/article/%d/%d/set-read", apd.FeedID, apd.ArticleId)),
			),
			P(
				A(Text("Back to top"), Href("#homepage")),
			),
		),
		Script(Src("/public/js/feeds.js")),
	)
}
