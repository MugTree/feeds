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
		LikeArticle(aps.FeedID, aps.ArticleId, aps.StarValue),
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

func LikeArticle(feedID int64, articleID int64, starsValue int64) Node {
	return Div(
		ID("star-value-bottom"),
		ds.On("click", fmt.Sprintf("@put('/article/%v/%v/like/%v')", feedID, articleID, starsValue)),
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
		comments = append(comments, Div(
			Data("note-id", strconv.FormatInt(i, 10)),
			Class("note-holder"),
			P(Text(note.CommentText)),
		))

	}
	return Aside(ID("article-notes"), Group(comments))
}

func EditComments(msn CommentsTemplateData) Node {
	forms := []Node{}
	for i := range msn.TotalPotentialCommentsCount {
		note, _ := msn.Comments[i]
		var elem Node

		if i == msn.NoteToEdit && msn.ShowTextArea {
			elem = Form(
				Class("note-edit"),
				Data("note-id", strconv.FormatInt(i, 10)),
				Textarea(
					Name("note-text"),
					Data("note-id", strconv.FormatInt(i, 10)),
					Text(note.CommentText),
				),
				Button(
					Text("Edit"),
					ds.On("click", fmt.Sprintf("@post('/article/%v/note/write/%v', {contentType: 'form'})", msn.ArticleID, msn.NoteToEdit)),
				),
			)
		} else {
			elem = Div(
				Class("note-holder"),
				Data("note-id", strconv.FormatInt(i, 10)),
				Text(note.CommentText),
			)
		}
		forms = append(forms, elem)
	}
	return Aside(ID("article-notes"), Group(forms))
}

// func TestComp() Node {
// 	cps := []Node{}

// 	for range 7 {
// 		cps = append(cps, Div(Text("blah")))
// 	}

// 	return Group(cps)
// }
