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

type TemplatePageProps struct {
	Title       string
	Description string
}

func TemplateNote(articleID int, noteContents string) Node {
	return Div(Class("note"),
		Textarea(ID("note-input")),
	)
}

func TemplateLayout(props TemplatePageProps, children ...Node) Node {

	return HTML5(HTML5Props{
		Title:       props.Title,
		Description: props.Description,
		Language:    "en",
		Head: []Node{
			Script(Src("/public/js/datastar.js"), Type("module")),
			Link(Rel("stylesheet"), Href("/public/css/main.css")),
		},
		Body: []Node{Div(ID("pagetop"),

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

/*
-----------------------------------
Homepage
-----------------------------------
*/

func TemplateHomePage(summaries []FeedSummary) Node {

	feedPagination := func(fsm FeedSummary) Node {
		links := []Node{}
		for i := range fsm.LinksRequired {
			pageNumber := i + 1
			links = append(links,
				A(
					ds.On("click", fmt.Sprintf("@get('/feed/%v/page/%v')", fsm.FeedID, pageNumber)),
					Text(strconv.FormatInt(pageNumber, 10)),
					Classes{"link": true, "underline": fsm.PageID == pageNumber},
				),
			)
		}
		return Group(links)
	}

	return Div(ID("homepage"),
		Div(ID("feeds"),
			Map(summaries, func(fs FeedSummary) Node {
				return Div(ID(fmt.Sprintf("feed-box-%v", fs.FeedID)),
					Class("feedbox"),
					H2(Text(fs.Name), ds.On("click", fmt.Sprintf("@get('/feed/%v/page/1')", fs.FeedID))),

					If(len(fs.Articles) > 0,
						Div(
							Map(fs.Articles,
								func(a db.SelectArticlesByFeedIDWithLimitRow) Node {

									return H3(A(Text(a.ArticleTitle), Href(fmt.Sprintf("/article/%v/view", a.ArticleID))))
								},
							),
							Div(feedPagination(fs)),
						),
					),
				)

			}),
		),
	)

}

// func TemplateTextAreaInput(notes []string) Node {

// 	textAreaVal := ""
// 	if len(notes) > 0 {
// 		fmt.Println("adding notes")
// 		textAreaVal = strings.Join(notes, "\n\n")
// 	}

// 	return Textarea(
// 		ds.Bind("notesText"),
// 		ds.On("input", "$complete = textAreaComplete(evt)"),
// 		ID("user_input"),
// 		Text(textAreaVal),
// 	)
// }

func TemplateArticlePage(apd ArticlePageData) Node {
	return Div(ID("article"), ds.Signals(map[string]any{"hideTitle": false}),

		Div(
			ID("article-top"),
			H2(
				Text(fmt.Sprintf("%v - %v", apd.FeedTitle, apd.PageTitle)),
				Data("show", "!$hideTitle"),
			),
			Ul(
				Li(Text(apd.ArticlePublished)),
				Li(Text(apd.FeedTitle), ds.On("click", "$hideTitle = !$hideTitle")),
				Li(TemplateLikeArticle(apd.ArticleId, apd.StarValue)),
			),
		),

		Section(Class("grid-parent"),
			Div(ID("article"),
				Raw(apd.PageContent),
				//Raw("<p>Para 1</p><p>Para 2</p><p>Para 3</p><p>Para 4</p><p>Para 5</p><p>Para 6</p><p>Para 7</p>"),
			),
			Div(
				Textarea(ID("editor"), Style("width: 90%; height: 100vh"),
					Value("Some value"),
				),
			),
			//partialViewComments(apd.CommentsTemplateData),
		),

		TemplateLikeArticle(apd.ArticleId, apd.StarValue),

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
	)
}

// func TemplateFeedBoxes(summaries []FeedSummary) Node {
// 	return Map(summaries, func(fs FeedSummary) Node {
// 		return TemplateFeedBox(fmt.Sprintf("@get('/feed/%v/page/1')", fs.FeedID), fs)
// 	})
// }

func TemplateLikeArticle(articleID int64, starsValue int64) Node {
	return Div(ID("star-value-bottom"), ds.On("click", fmt.Sprintf("@put('/article/%v/like/%v')", articleID, starsValue)),

		Text("Starred:"),

		Img(
			Width("60px"),
			Src(fmt.Sprintf("/public/img/%v-star.png", starsValue)),
		),
	)
}

func TemplateBasicTextInput(labelText string, labelFor string, bindVal string, value string, notValid bool) Node {

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

func TemplateFeedsPageAdminForm(td FeedFormTemplateData) Node {

	return Form(ID("admin-form"),
		ds.On("submit", fmt.Sprintf(`@put('/admin/feed/%v/update'), {contentType: 'form'}`, td.Feed.ID)),

		TemplateBasicTextInput("Title", "FeedName", "feed-name", td.Feed.Title, false),
		TemplateBasicTextInput("Feed Url", "FeedUrl", "feed-url", td.Feed.Url, false),
		TemplateBasicTextInput("CSS Container", "CssContainer", "css-sel-container", td.Feed.CssSelContainer, false),
		TemplateBasicTextInput("CSS Selector start", "CSSStart", "css-sel-start", td.Feed.CssSelStart, false),
		TemplateBasicTextInput("CSS Selector stop", "CSSStop", "css-sel-stop", td.Feed.CssSelStop, false),

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

					return partialOptionIsSelected(
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

func partialOptionIsSelected(txt string, selValue string, val string) Node {
	return Option(
		Value(val),
		Text(txt),
		If(selValue == val, Attr("selected")),
	)
}

func partialListFeeds(feeds []db.Feed) Node {
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
