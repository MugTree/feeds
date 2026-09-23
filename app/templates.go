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

type pageProps struct {
	Title       string
	Description string
}

func TemplateLayout(props pageProps, children ...Node) Node {

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
			Div(ID("pagetop"),

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

func TemplateHomePage(summaries []mpdFeedSummary) Node {
	return Div(ID("homepage"),
		Div(
			ID("feeds"),
			TemplateFeedBoxes(summaries),
		),
		Div(
			ID("article"),
		),
	)
}

func TemplateTextAreaInput(notes []string) Node {

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

// func pageGamePlay(article db.Article, articleParagraphs [][]ArticleParagraph, pageParagraphs []ArticleParagraph, pageNumber int, notes []string, lastPage int) Node {

// 	showNext := func(pageNumber int) bool {
// 		return pageNumber < lastPage
// 	}
// 	showPreviousLink := func(pageNumber int) bool {
// 		return pageNumber > 1
// 	}

// 	return Div(
// 		ID("game"),
// 		ds.Signals(
// 			map[string]any{
// 				"paragraphCount": len(pageParagraphs),
// 				"complete":       false,
// 				"notes":          "",
// 			}),
// 		Class("grid-parent"),
// 		Div(
// 			Class("paragraphs"),
// 			Map(pageParagraphs, func(p ArticleParagraph) Node {
// 				return P(
// 					Text(p.Text),
// 				)
// 			}),

// 			Div(
// 				If(showPreviousLink(pageNumber),
// 					A(
// 						Text("< Previous chunk"),
// 						Href(fmt.Sprintf("/game/article/%v/page/%v", article.ID, pageNumber-1)),
// 					)),
// 				Span(
// 					Text("|")),
// 				If(showNext(pageNumber),
// 					A(
// 						Text("Next chunk >"),
// 						Href(fmt.Sprintf("/game/article/%v/page/%v", article.ID, pageNumber+1)),
// 					),
// 				),
// 			),
// 		),
// 		Div(
// 			TemplateTextAreaInput(notes),
// 			Button(
// 				ds.Show("$complete == true"),
// 				Text("save"),
// 				ds.On("click", fmt.Sprintf(`@put("/game/article/%v/page/%v")`, article.ID, pageNumber)),
// 			),
// 		),
// 		//		notesPanel(notes),
// 		Div(ID("conclusion"),
// 			Map(articleParagraphs, func(pp []ArticleParagraph) Node {
// 				return Map(pp, func(p ArticleParagraph) Node {
// 					return P(Text(p.Text))
// 				})
// 			}),
// 		),
// 		Script(Src("/public/js/feeds.js")),
// 		Script(Raw("console.log('loading...'); equaliseHeights()")),
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

		Section(
			Class("editor"),
			ID("editor"),
			ds.Signals(
				map[string]any{
					//"HasBeenRead":                  apd.ArticleHasBeenRead(),
					"HasScrolledToBottomOfArticle": false},
			),
			Raw(apd.PageContent),
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

		Script(Src("/public/js/feeds.js")),
	)
}

func TemplateFeedBoxes(summaries []mpdFeedSummary) Node {
	return Map(summaries, func(fs mpdFeedSummary) Node {
		return TemplateFeedBox(fmt.Sprintf("@get('/feed/%v/page/1')", fs.FeedID), fs)
	})
}

func TemplateFeedBox(link string, fsm mpdFeedSummary) Node {

	pageLinks := func(fsm mpdFeedSummary) Node {
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

	return Div(
		Class("feedbox"),
		ID(fmt.Sprintf("feed-box-%v", fsm.FeedID)),
		H2(Text(fsm.Name), ds.On("click", link)),
		If(len(fsm.Articles) > 0,
			Div(
				Map(fsm.Articles,
					func(a db.SelectArticlesByFeedIDWithLimitRow) Node {
						articleURL := fmt.Sprintf("@get('/article/%v/view')", a.ArticleID)
						return H3(Text(a.ArticleTitle), ds.On("click", articleURL))
					},
				),
				Div(pageLinks(fsm)),
			),
		),
	)
}

func TemplateLikeArticle(articleID int64, starsValue int64) Node {
	return Div(
		ID("star-value-bottom"),
		ds.On("click", fmt.Sprintf("@put('/article/%v/like/%v')", articleID, starsValue)),
		Text("Starred:"),
		Img(
			Width("60px"),
			Src(fmt.Sprintf("/public/img/%v-star.png", starsValue)),
		),
	)
}

// func partialViewComments(mns mpdCommentsData) Node {
// 	comments := []Node{}
// 	for i := range mns.TotalPotentialCommentsCount {
// 		note, _ := mns.Comments[i]
// 		comments = append(comments,
// 			Div(
// 				Data("comment-id", strconv.FormatInt(i, 10)),
// 				Class("comment-holder"),
// 				P(Text(note.CommentText)),
// 			))

// 	}
// 	return Aside(ID("article-notes"), Group(comments))
// }

// func partialWriteComments(msn mpdCommentsData) Node {
// 	forms := []Node{}
// 	for i := range msn.TotalPotentialCommentsCount {
// 		note, _ := msn.Comments[i]
// 		var elem Node

// 		if i == msn.NoteToEdit && msn.ShowTextArea {
// 			elem = Form(
// 				Class("comment-edit"),
// 				Data("comment-id", strconv.FormatInt(i, 10)),
// 				Textarea(
// 					Name("comment-text"),
// 					Data("comment-id", strconv.FormatInt(i, 10)),
// 					Text(note.CommentText),
// 				),
// 				Button(
// 					Text("Edit"),
// 					ds.On("click", fmt.Sprintf("@post('/article/%v/comment/%v/write', {contentType: 'form'})", msn.ArticleID, msn.NoteToEdit)),
// 				),
// 			)
// 		} else {
// 			elem = Div(
// 				Class("comment-holder"),
// 				Data("comment-id", strconv.FormatInt(i, 10)),
// 				Text(note.CommentText),
// 			)
// 		}
// 		forms = append(forms, elem)
// 	}
// 	return Aside(ID("article-notes"), Group(forms))
// }

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

	return Form(
		ds.On("submit", fmt.Sprintf(`@put('/admin/feed/%v/update'), {contentType: 'form'}`, td.Feed.ID)),
		ID("admin-form"),

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
