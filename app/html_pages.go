package app

import (
	"fmt"
	"strconv"

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
					Href(fmt.Sprintf("/game/magpie/%v/0", a.ID)),
				),
			)
		}),
	)
}

func pageGamePlay(article db.Article, chunk []mpdParagraph, index int) Node {

	hasSummary := func(index int) bool {
		return index == 2
	}

	return Div(Class("flex"),
		If(len(chunk) > 0, Div(
			Ul(
				Class("paras"),
				Map(chunk, func(p mpdParagraph) Node {
					id := strconv.Itoa(p.Index)
					return Li(
						P(
							ds.On("click", "!TODO - showRelatedPara()"),
							Data("paragraph", id),
							If(p.Type == "blockquote", Class("block-quote")),
							Text(p.Text),
						),
						Textarea(
							ID(id),
							Data("editor", id),
							ds.On("click", `@put('/comment')`),
						),
						If(
							hasSummary(p.Index),
							Textarea(
								ID(fmt.Sprintf("summary-%v", id)),
								Data("summary", id),
								ds.On("click", `@put("/comment")`),
							),
						),
					)
				}),
			),
			Button(
				Text("Next chunk >"),
				ds.On("click", fmt.Sprintf(`@get("/game/magpie/%v")`, index)),
			),
		),
		), Div(
			Class("raw"),
			Raw(article.ArticleContent),
		),
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
