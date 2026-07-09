package main

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

// depth first traversal
func feedsBuildStructuredDocument(htmlInput string) (*Document, error) {

	// create teh tree
	root, err := html.Parse(strings.NewReader(htmlInput))
	if err != nil {
		return nil, err
	}

	// structure to contain the tree plus an index of all of the text nodes
	doc := &Document{
		Root:      root,
		TextIndex: make(map[int]*TextNode),
	}

	id := 0

	// written this way as we recurse
	// basic depth first traversal
	var walk func(*html.Node)

	walk = func(n *html.Node) {

		// on a non nil node step in
		for c := n.FirstChild; c != nil; {

			// basic idea
			// -----------------------------------------------
			// next := c.NextSibling   // remember my brother
			// walk(c)                 // visit my children
			// c = next                // now visit my brother

			next := c.NextSibling

			if c.Type == html.TextNode {

				// does the node have any content
				// the preprocessing shoudl have stripped empty nodes out
				if strings.TrimSpace(c.Data) != "" {

					// create a new node with out markup
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

					// step out to the text nodes parent node
					parent := c.Parent

					// remove the current text node
					parent.RemoveChild(c)

					// append it to the wrapper we have created
					wrapper.AppendChild(c)

					// insert before using the reference we made earlier
					parent.InsertBefore(wrapper, next)

					// add out newly created to the TextIndex for later reference
					doc.TextIndex[id] = &TextNode{
						ID:      id,
						Wrapper: wrapper,
					}

					// step up the counter - this is the key for the index and
					// also used in the html so we can get hold of the node in the front end more easily
					id++
				}

				// step forward to the next iteration
				c = next
				continue
			}

			// recursion
			if c.FirstChild != nil {
				walk(c)
			}

			// this allows us to step across the document and not just drill down
			c = next
		}
	}

	walk(root)

	return doc, nil
}

//
// Apply annotations
//

func feedsApplyAnnotations(doc *Document, annotations []Annotation) {

	for _, a := range annotations {

		// get the start and end points in the text index
		start := doc.TextIndex[a.Start.TextID]
		end := doc.TextIndex[a.End.TextID]

		// basic check
		if start == nil || end == nil {
			continue
		}

		feedsApplyAnnotation(start.Wrapper, a)
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

func feedsApplyAnnotation(wrapper *html.Node, a Annotation) {

	text := _getText(wrapper)

	// do the offsets look wrong - too small / large
	// is the start greater than the end
	if a.Start.Offset < 0 ||
		a.End.Offset > len(text) ||
		a.Start.Offset >= a.End.Offset {
		return
	}

	// get the first part
	before := text[:a.Start.Offset]
	// get the middle part
	selected := text[a.Start.Offset:a.End.Offset]
	// get the end part
	after := text[a.End.Offset:]

	// remove current contents
	for wrapper.FirstChild != nil {
		wrapper.RemoveChild(wrapper.FirstChild)
	}

	// append part of the text that doesnt need wrapping
	if before != "" {
		wrapper.AppendChild(
			&html.Node{
				Type: html.TextNode,
				Data: before,
			},
		)
	}

	// append the span node..
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

	// append the contents "selected" ...
	highlight.AppendChild(
		&html.Node{
			Type: html.TextNode,
			Data: selected,
		},
	)

	// add that back to the outer wrapper node
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

//
// Extract text from a wrapper
//

func _getText(n *html.Node) string {

	var b strings.Builder

	var walk func(*html.Node)

	walk = func(n *html.Node) {

		// get the text a GTFO
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			return
		}

		// keep going if required
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

func feedsRenderHTMLStr(doc *Document) (string, error) {

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

	doc, err := feedsBuildStructuredDocument(rawHTML)

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
				Offset: 7,
			},
		},
	}

	//
	// 4. Apply annotations
	//

	feedsApplyAnnotations(doc, annotations)

	//
	// 5. Render response
	//

	output, err := feedsRenderHTMLStr(doc)

	if err != nil {
		panic(err)
	}

	fmt.Println(output)
}
