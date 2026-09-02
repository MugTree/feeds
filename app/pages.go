package app

import (
	"fmt"
	"strconv"

	. "maragu.dev/gomponents"
	ds "maragu.dev/gomponents-datastar"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type pageProps struct {
	Title       string
	Description string
}

func Layout(props pageProps, children ...Node) Node {

	return HTML5(HTML5Props{
		Title:       props.Title,
		Description: props.Description,
		Language:    "en",
		Head: []Node{
			Script(Src("/public/js/datastar.js"), Type("module")),
			// Link(Rel("stylesheet"), Href("/public/css/main.css")),
		},
		Body: []Node{Class(""),
			Div(
				ID("pagetop"),
				Header(
					A(
						Href("/"),
						Class("home-link"),
						Text("Home"),
					),
					Text("|"),
					A(
						Href("/admin"),
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

func getFeedUrl(feedID int64) string {
	return fmt.Sprintf("@get('/feed/%v/page/1')", feedID)
}

func PageHome(summaries []FeedSummary) Node {
	return Div(
		ID("homepage"),
		Div(
			ID("feeds"),
			FeedBoxes(summaries),
		),
		Div(
			ID("article"),
		),
	)
}

func FeedBoxes(summaries []FeedSummary) Node {
	return Map(summaries, func(fs FeedSummary) Node {
		return FeedBox(getFeedUrl(fs.FeedID), fs)
	})
}

func FeedBox(link string, fsm FeedSummary) Node {
	pagination := func(fsm FeedSummary) Node {
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
		ID(fmt.Sprintf("feed-box-%v", fsm.FeedID)),
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
				Li(Text(aps.ArticlePublished)),
				Li(Text(aps.FeedTitle), ds.On("click", "$hideTitle = !$hideTitle")),
				Li(LikeArticle(aps.ArticleId, aps.StarValue)),
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
			ViewComments(aps.CommentsTemplateData),
		),
		LikeArticle(aps.ArticleId, aps.StarValue),
		Section(
			H3(ds.On("intersect", "$HasScrolledToBottomOfArticle = true")),
			Button(
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

func LikeArticle(articleID int64, starsValue int64) Node {
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

func ViewComments(mns CommentsTemplateData) Node {
	comments := []Node{}
	for i := range mns.TotalPotentialCommentsCount {
		note, _ := mns.Comments[i]
		comments = append(comments,
			Div(
				Data("comment-id", strconv.FormatInt(i, 10)),
				Class("comment-holder"),
				P(Text(note.CommentText)),
			))

	}
	return Aside(ID("article-notes"), Group(comments))
}

func WriteComments(msn CommentsTemplateData) Node {
	forms := []Node{}
	for i := range msn.TotalPotentialCommentsCount {
		note, _ := msn.Comments[i]
		var elem Node

		if i == msn.NoteToEdit && msn.ShowTextArea {
			elem = Form(
				Class("comment-edit"),
				Data("comment-id", strconv.FormatInt(i, 10)),
				Textarea(
					Name("comment-text"),
					Data("comment-id", strconv.FormatInt(i, 10)),
					Text(note.CommentText),
				),
				Button(
					Text("Edit"),
					ds.On("click", fmt.Sprintf("@post('/article/%v/comment/%v/write', {contentType: 'form'})", msn.ArticleID, msn.NoteToEdit)),
				),
			)
		} else {
			elem = Div(
				Class("comment-holder"),
				Data("comment-id", strconv.FormatInt(i, 10)),
				Text(note.CommentText),
			)
		}
		forms = append(forms, elem)
	}
	return Aside(ID("article-notes"), Group(forms))
}
