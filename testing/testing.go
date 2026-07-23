package main

import (
	"fmt"
	"strings"

	"github.com/starfederation/datastar-go/datastar"
	"golang.org/x/net/html"
)

func feedsEnrichSantitizedHTMLWithEvents(input string) (*html.Node, error) {

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return nil, err
	}

	var enrichHTML func(n *html.Node)

	enrichHTML = func(n *html.Node) {

		for c := n.FirstChild; c != nil; {

			next := c.NextSibling

			attrs := c.Attr

			for _, v := range attrs {

				if v.Key == "data-block-id" {

					attrs = append(attrs, html.Attribute{
						Key: "data-on:click",
						Val: datastar.PutSSE("/url/%v", v.Val),
					})
				}

			}

			// update the attributes
			c.Attr = attrs

			if c.FirstChild != nil {
				enrichHTML(c)
			}

			c = next
		}

	}

	enrichHTML(doc)
	return doc, nil
}

func feedsTransformSanitizedToArticle(doc *html.Node) (*html.Node, error) {

	var body *html.Node

	var walk func(*html.Node)

	walk = func(n *html.Node) {
		if body != nil {
			return
		}

		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	if body == nil {
		return nil, fmt.Errorf("body element not found")
	}

	article := &html.Node{
		Type: html.ElementNode,
		Data: "article",
	}

	// Move every child from <body> into <article>.
	for body.FirstChild != nil {
		child := body.FirstChild
		body.RemoveChild(child)
		article.AppendChild(child)
	}

	return article, nil
}

func main() {

	str := `<html><head></head><body><p data-block-id="0"><em>Come on boys, stay calm. Everyone stay calm. The principle, the main thing, let’s stay calm boys, stay calm, composure. Let’s just think about playing, stay calm. Let’s forget about everything, eh? Let’s just play, let’s just focus on playing, stay calm. Come on.</em></p><p data-block-id="1">Those words from Messi show how it appears that he and the rest of the Argentina squad learned right before the game that the wind at their backs they’d been enjoying was going to be benefiting Spain. It also explains why they turned their backs on the trophy presentation ceremony; most of the squad are serious Catholics. Why? Well, as one man has demonstrated, it appears the game itself, indeed, the entire tournament, was a Clown World ritual that was set up more than two decades ago.</p><div><figure><img src="https://voxday.net/wp-content/uploads/2026/07/image-11.png" alt=""/></figure></div><p data-block-id="2">There is a lot more than that; the Economist cover is not proof of anything, but it is consistent with the theme, and, of course, we know that FIFA is wholly owned by Clown World, as is most of the so-called “entertainment” industry. And the humiliation ritual of Donald Trump at the end would also appear to have been staged as part of the whole act. Whatever was going on, it was almost certainly a little darker than WWE-style scripting.</p><p data-block-id="3"><em><a href="https://socialgalactic.com/micropost/1e165433-4a0c-43ca-8c5d-370f3d54c165">DISCUSS ON SG</a></em></p></body></html>`
	htmlDoc, err := feedsEnrichSantitizedHTMLWithEvents(str)
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	article, err := feedsTransformSanitizedToArticle(htmlDoc)
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	var b strings.Builder
	html.Render(&b, article)

	fmt.Println(b.String())

}
