package app

import (
	"fmt"
	"strings"

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

func textAreaInput(notes []string) Node {

	textAreaVal := ""
	if len(notes) > 0 {
		fmt.Println("adding notes")
		textAreaVal = strings.Join(notes, "\n\n")
	}

	return Textarea(
		ds.Bind("notesText"),
		ds.On("input", "$complete = textAreaComplete(evt)"),
		ID("user_input"),
		Text(textAreaVal),
	)
}

func pageGamePlay(article db.Article, articleParagraphs [][]ArticleParagraph, pageParagraphs []ArticleParagraph, pageNumber int, notes []string, lastPage int) Node {

	showNext := func(pageNumber int) bool {
		return pageNumber < lastPage
	}
	showPreviousLink := func(pageNumber int) bool {
		return pageNumber > 1
	}

	return Div(
		ID("game"),
		ds.Signals(
			map[string]any{
				"paragraphCount": len(pageParagraphs),
				"complete":       false,
				"notes":          "",
			}),
		Class("grid-parent"),
		Div(
			Class("paragraphs"),
			Map(pageParagraphs, func(p ArticleParagraph) Node {
				return P(
					Text(p.Text),
				)
			}),

			Div(
				If(showPreviousLink(pageNumber),
					A(
						Text("< Previous chunk"),
						Href(fmt.Sprintf("/game/article/%v/page/%v", article.ID, pageNumber-1)),
					)),
				Span(
					Text("|")),
				If(showNext(pageNumber),
					A(
						Text("Next chunk >"),
						Href(fmt.Sprintf("/game/article/%v/page/%v", article.ID, pageNumber+1)),
					),
				),
			),
		),
		Div(
			textAreaInput(notes),
			Button(
				ds.Show("$complete == true"),
				Text("save"),
				ds.On("click", fmt.Sprintf(`@put("/game/article/%v/page/%v")`, article.ID, pageNumber)),
			),
		),
		//		notesPanel(notes),
		Div(ID("conclusion"),
			Map(articleParagraphs, func(pp []ArticleParagraph) Node {
				return Map(pp, func(p ArticleParagraph) Node {
					return P(Text(p.Text))
				})
			}),
		),
		Script(Src("/public/js/feeds.js")),
		Script(Raw("console.log('loading...'); equaliseHeights()")),
	)
}

// func pageArticle(apd mpdArticlePageData) Node {
// 	return Div(
// 		ID("article"),
// 		ds.Signals(map[string]any{"hideTitle": false}),
// 		Div(
// 			ID("article-top"),
// 			H2(
// 				Text(fmt.Sprintf("%v - %v", apd.FeedTitle, apd.PageTitle)),
// 				Data("show", "!$hideTitle"),
// 			),
// 			Ul(
// 				Li(Text(apd.ArticlePublished)),
// 				Li(Text(apd.FeedTitle), ds.On("click", "$hideTitle = !$hideTitle")),
// 				Li(partialLikeArticle(apd.ArticleId, apd.StarValue)),
// 			),
// 		),
// 		Section(
// 			Class("editor"),
// 			ID("editor"),
// 			ds.Signals(
// 				map[string]any{
// 					"HasBeenRead":                  apd.ArticleHasBeenRead(),
// 					"HasScrolledToBottomOfArticle": false},
// 			),
// 			Raw(apd.PageContent),
// 			//partialViewComments(apd.CommentsTemplateData),
// 		),
// 		partialLikeArticle(apd.ArticleId, apd.StarValue),
// 		Section(
// 			H3(ds.On("intersect", "$HasScrolledToBottomOfArticle = true")),
// 			Button(
// 				Class("have-read-article"),
// 				Text("Mark as read"),
// 				Data("show", "$HasScrolledToBottomOfArticle && !$HasBeenRead"),
// 				ds.On("click", fmt.Sprintf("/article/%d/%d/set-read", apd.FeedID, apd.ArticleId)),
// 			),
// 			P(
// 				A(Text("Back to top"), Href("#homepage")),
// 			),
// 		),
// 		Script(Src("/public/js/feeds.js")),
// 	)
// }
