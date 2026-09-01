package app

import (
	. "maragu.dev/gomponents"
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
			Script(Src("/public/js/feeds.js")),
			Link(Rel("stylesheet"), Href("/public/css/main.css")),
		},
		Body: []Node{Class(""),
			Div(Class(""),
				Group(children),
			),
		},
	})
}
