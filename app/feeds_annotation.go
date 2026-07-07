package app

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

//
// Models
//

type SelectionPoint struct {
	TextID int `json:"textId"`
	Offset int `json:"offset"`
}

type Annotation struct {
	ID    string         `json:"id"`
	Start SelectionPoint `json:"start"`
	End   SelectionPoint `json:"end"`
}

type TextNode struct {
	ID      int
	Wrapper *html.Node
}

type Document struct {
	Root      *html.Node
	TextIndex map[int]*TextNode
}

//
// Build document
//

func feedsAnnotationBuildDocument(htmlInput string) (*Document, error) {

	root, err := html.Parse(strings.NewReader(htmlInput))
	if err != nil {
		return nil, err
	}

	doc := &Document{
		Root:      root,
		TextIndex: make(map[int]*TextNode),
	}

	id := 0

	var walk func(*html.Node)

	walk = func(n *html.Node) {

		for c := n.FirstChild; c != nil; {

			next := c.NextSibling

			if c.Type == html.TextNode {

				if strings.TrimSpace(c.Data) != "" {

					wrapper := &html.Node{
						Type: html.ElementNode,
						Data: "span",
						Attr: []html.Attribute{
							{
								Key: "data-text-id",
								Val: strconv.Itoa(id),
							},
						},
					}

					parent := c.Parent

					parent.RemoveChild(c)

					wrapper.AppendChild(c)

					parent.InsertBefore(wrapper, next)

					doc.TextIndex[id] = &TextNode{
						ID:      id,
						Wrapper: wrapper,
					}

					id++
				}

				c = next
				continue
			}

			if c.FirstChild != nil {
				walk(c)
			}

			c = next
		}
	}

	walk(root)

	return doc, nil
}

//
// Apply annotations
//

func feedsAnnotationApplyMultiple(doc *Document, annotations []Annotation) {

	for _, a := range annotations {

		start := doc.TextIndex[a.Start.TextID]
		end := doc.TextIndex[a.End.TextID]

		if start == nil || end == nil {
			continue
		}

		feedsAnnotationApplySingle(start.Wrapper, a)
	}
}

//
// Apply single annotation
//
// This version handles the common case:
// annotation exists entirely inside one text-id span.
//
// Multi-node selections are handled by extending this pattern.
//

func feedsAnnotationApplySingle(wrapper *html.Node, a Annotation) {

	text := _getText(wrapper)

	if a.Start.Offset < 0 ||
		a.End.Offset > len(text) ||
		a.Start.Offset >= a.End.Offset {
		return
	}

	before := text[:a.Start.Offset]
	selected := text[a.Start.Offset:a.End.Offset]
	after := text[a.End.Offset:]

	// remove current contents
	for wrapper.FirstChild != nil {
		wrapper.RemoveChild(wrapper.FirstChild)
	}

	if before != "" {
		wrapper.AppendChild(
			&html.Node{
				Type: html.TextNode,
				Data: before,
			},
		)
	}

	highlight := &html.Node{
		Type: html.ElementNode,
		Data: "span",
		Attr: []html.Attribute{
			{
				Key: "class",
				Val: "highlight",
			},
			{
				Key: "data-id",
				Val: a.ID,
			},
		},
	}

	highlight.AppendChild(
		&html.Node{
			Type: html.TextNode,
			Data: selected,
		},
	)

	wrapper.AppendChild(highlight)

	if after != "" {
		wrapper.AppendChild(
			&html.Node{
				Type: html.TextNode,
				Data: after,
			},
		)
	}
}

// remove extraneous tags and attributes. script, style, template, comments and attributes on a per tag basis
func feedsAnnotationSanitizeHTMLForStorage(input string) (*html.Node, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return nil, err
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

			allowed := allowedAttrs(c.Data)

			attrs := c.Attr[:0] // reuse underlying array
			for _, v := range c.Attr {
				if _, ok := allowed[v.Key]; ok {
					attrs = append(attrs, v)
				}
			}
			c.Attr = attrs

			// remove any empty tags - artifacts if editors perhaps
			if c.Type == html.ElementNode &&
				len(c.Attr) == 0 &&
				c.FirstChild == nil {

				switch strings.ToLower(c.Data) {
				case "div", "span", "p":
					n.RemoveChild(c)
				}
			}

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
	return doc, nil

}

//
// Extract text from a wrapper
//

func _getText(n *html.Node) string {

	var b strings.Builder

	var walk func(*html.Node)

	walk = func(n *html.Node) {

		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)

	return b.String()
}

//
// Render
//

func _render(doc *Document) (string, error) {

	var b strings.Builder

	err := html.Render(&b, doc.Root)

	if err != nil {
		return "", err
	}

	return b.String(), nil
}

//
// Example workflow
//

func main() {

	rawHTML := `
	<html>
	<body>
	<p>Hello world this is a test</p>
	<p>Another paragraph</p>
	</body>
	</html>
	`

	//
	// 1. Sanitise here
	//
	// cleanHTML, _ := sanitizeHTML(rawHTML)

	//
	// 2. Build document
	//

	doc, err := feedsAnnotationBuildDocument(rawHTML)

	if err != nil {
		panic(err)
	}

	//
	// 3. User submits annotation
	//

	annotations := []Annotation{

		{
			ID: "abc123",

			Start: SelectionPoint{
				TextID: 0,
				Offset: 6,
			},

			End: SelectionPoint{
				TextID: 0,
				Offset: 11,
			},
		},
	}

	//
	// 4. Apply annotations
	//

	feedsAnnotationApplyMultiple(doc, annotations)

	//
	// 5. Render response
	//

	output, err := _render(doc)

	if err != nil {
		panic(err)
	}

	fmt.Println(output)
}
