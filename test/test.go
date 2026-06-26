package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func sanitizeHTML(input string) (string, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", err
	}

	var cleanHTML func(n *html.Node)
	cleanHTML = func(n *html.Node) {

		shouldRemoveElement := func(n *html.Node) bool {
			if n.Type != html.ElementNode {
				return false
			}

			switch strings.ToLower(n.Data) {
			case "script", "noscript", "style", "template":
				return true
			default:
				return false
			}
		}

		for c := n.FirstChild; c != nil; {

			next := c.NextSibling

			if shouldRemoveElement(c) {
				n.RemoveChild(c)
				c = next
				continue
			}

			if c.Type == html.CommentNode {
				n.RemoveChild(c)
				c = next
				continue
			}

			if c.FirstChild != nil {
				cleanHTML(c)
			}

			c = next

		}
	}

	cleanHTML(doc)

	var b strings.Builder
	err = html.Render(&b, doc)
	if err != nil {
		return "", err
	}

	return b.String(), nil

}

func main() {

	input := `<html><head></head><body><script></script><!----><style>body{sfgsf}</style></body><html>`

	htmlStr, err := sanitizeHTML(input)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(htmlStr)

}
