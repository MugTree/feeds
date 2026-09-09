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

func partialFeedBoxes(summaries []mpdFeedSummary) Node {
	return Map(summaries, func(fs mpdFeedSummary) Node {
		return partialFeedBox(fmt.Sprintf("@get('/feed/%v/page/1')", fs.FeedID), fs)
	})
}

func partialFeedBox(link string, fsm mpdFeedSummary) Node {

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
					func(a mpdEnrichedArticle) Node {
						articleURL := fmt.Sprintf("@get('/article/%v/view')", a.Article.ArticleID)
						return H3(Text(a.Article.ArticleTitle), ds.On("click", articleURL))
					},
				),
				Div(pageLinks(fsm)),
			),
		),
	)
}

func partialLikeArticle(articleID int64, starsValue int64) Node {
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

func partialViewComments(mns mpdCommentsData) Node {
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

func partialWriteComments(msn mpdCommentsData) Node {
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
