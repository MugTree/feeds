package main

import (
	"net/http"

	"maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

func Page() gomponents.Node {
	return html.HTML(
		html.Body(
			html.H1(
				gomponents.Text("Hello gomponents"),
			),
			html.P(
				gomponents.Text("This is my first component."),
			),
		),
	)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Page().Render(w)
	})

	http.ListenAndServe(":8080", nil)
}
