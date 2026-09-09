package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mugtree/feeds/app/db"
	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type PageProps struct {
	Title       string
	Description string
}

func htmlLayout(props PageProps, children ...Node) Node {

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

func htmlHomePage(summaries []FeedSummary) Node {
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

func htmlArticlePage(aps ArticlePageTemplateData) Node {
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

func partialBasicTextInput(labelText string, labelFor string, bindVal string, value string, notValid bool) Node {

	return Div(
		Label(
			For(labelFor),
			Text(labelText),
		),
		Input(
			ID(labelFor),
			// Placeholder(labelText),
			ds.Bind(bindVal),
			Type("text"),
			Value(value),
			Attr("aria-invalid", strconv.FormatBool(notValid)),
		),
	)
}

type FeedFormTemplateData struct {
	ButtonText string
	UrlAction  string
	Feed       db.Feed
	InitialRun bool
}

func partialFeedsAdminForm(td FeedFormTemplateData) Node {

	return Form(
		ds.On("submit", fmt.Sprintf(`@put('/admin/feed/%v/update'), {contentType: 'form'}`, td.Feed.ID)),
		ID("admin-form"),

		partialBasicTextInput("Title", "FeedName", "feed-name", td.Feed.Title, false),
		partialBasicTextInput("Feed Url", "FeedUrl", "feed-url", td.Feed.Url, false),
		partialBasicTextInput("CSS Container", "CssContainer", "css-sel-container", td.Feed.CssSelContainer, false),
		partialBasicTextInput("CSS Selector start", "CSSStart", "css-sel-start", td.Feed.CssSelStart, false),
		partialBasicTextInput("CSS Selector stop", "CSSStop", "css-sel-stop", td.Feed.CssSelStop, false),

		Div(
			Label(
				For("Strategy"),
				Text("Strategy"),
			),
			Select(
				ID("Strategy"),
				Name("Strategy"),
				ds.Bind("html-extraction-strategy"),
				Map([]string{defaultStrategyVal, "No Clip", "Clip End", "Clip Between"}, func(name string) Node {

					val := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
					selected := td.Feed.HtmlExtractionStrategy

					return OptionIsSelected(
						name,
						selected,
						val,
					)
				}),
			),
		),
		Div(
			Button(
				Text(td.ButtonText),
			),
		),
	)
}

const defaultStrategyVal string = "-- Set a strategy --"

func OptionIsSelected(txt string, selValue string, val string) Node {
	return Option(
		Value(val),
		Text(txt),
		If(selValue == val, Attr("selected")),
	)
}

func ListFeeds(feeds []db.Feed) Node {
	return Ul(
		Map(feeds, func(f db.Feed) Node {
			return Li(
				A(
					Href(fmt.Sprintf("/admin/feed/%v/view", f.ID)),
					Text(f.Title),
				),
			)
		},
		),
	)
}
