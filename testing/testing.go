package main

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func feedsSanitizeAndAnnotateHTMLForStorage(input string) (*html.Node, int64, error) {

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return nil, 0, err
	}

	allowedAttrs := func(tag string) map[string]struct{} {
		switch tag {
		case "a":
			return map[string]struct{}{
				"href": {},
			}
		case "img":
			return map[string]struct{}{
				"src": {},
				"alt": {},
			}
		case "td", "th":
			return map[string]struct{}{
				"colspan": {},
				"rowspan": {},
			}
		default:
			return nil
		}
	}

	isBlockElement := func(tag string) bool {
		switch tag {
		case "p",
			//"h1",
			//"h2",
			//"h3",
			//"h4",
			//"div",
			"figure",
			"blockquote",
			"ul",
			"ol",
			"table":
			return true
		default:
			return false
		}
	}

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

	clickableBlockID := 0

	var cleanHTML func(n *html.Node, ancestorIsBlock bool)

	cleanHTML = func(n *html.Node, ancestorIsBlock bool) {

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

			childAncestorIsBlock := ancestorIsBlock

			if c.Type == html.ElementNode {

				allowed := allowedAttrs(strings.ToLower(c.Data))

				attrs := c.Attr[:0]

				for _, v := range c.Attr {

					if _, ok := allowed[v.Key]; ok {
						attrs = append(attrs, v)
					}
				}

				c.Attr = attrs

				if isBlockElement(strings.ToLower(c.Data)) && !ancestorIsBlock {

					c.Attr = append(c.Attr, html.Attribute{
						Key: "data-block-id",
						Val: strconv.Itoa(clickableBlockID),
					})

					clickableBlockID++
					childAncestorIsBlock = true
				}
			}

			if c.FirstChild != nil {
				cleanHTML(c, childAncestorIsBlock)
			}

			if c.Type == html.ElementNode &&
				len(c.Attr) == 0 &&
				c.FirstChild == nil {

				switch strings.ToLower(c.Data) {
				case "div", "span", "p":
					n.RemoveChild(c)
					c = next
					continue
				}
			}

			c = next
		}
	}

	cleanHTML(doc, false)

	// disambiguate the counter from the count
	clickableBlockCount := int64(clickableBlockID)

	return doc, clickableBlockCount, nil
}

func main() {

	str := `<html><head></head><body>
	<blockquote><p class="some-crap">a</p></blockquote>
	<p>b</p>
	<p>c</p> 
	<p>d</p>
	<p>f</p></body></html>`

	doc, blockCount, err := feedsSanitizeAndAnnotateHTMLForStorage(str)
	if err != nil {
		fmt.Print(err.Error())
		return
	}

	fmt.Printf("blockCount: %v\n", blockCount)
	var b strings.Builder
	html.Render(&b, doc)

	fmt.Println(b.String())

}
