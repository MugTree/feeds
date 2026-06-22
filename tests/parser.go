package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func main() {

	// replace newlines, tabs and spaces...
	// ------------------------------------------
	re := strings.NewReplacer("\n", "", "\t", "")

	str := `<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>Document</title>
		</head>
		<body>
			<h1>Hey!</h1>
			<p>blah blah</p>
			<ul>
				<li>a</li>
				<li>b</li>
				<li>c</li>
			</ul>
		</body>
	</html>`

	ns := re.Replace(str)

	an := []Annotation{{Start: 8, End: 12, ID: "1"}}

	html, _ := applyAnnotations(ns, an)
	fmt.Println(html)

}

type Annotation struct {
	Start int
	End   int
	ID    string
}

type TextSegment struct {
	Node  *html.Node
	Start int
	End   int
}

// ---------- OVERLAP CHECK ----------

func applyAnnotations(htmlInput string, annotations []Annotation) (string, error) {

	// get the html tree
	doc, err := html.Parse(strings.NewReader(htmlInput))
	if err != nil {
		return "", err
	}

	var segments []TextSegment
	offset := 0

	var buildIndex func(*html.Node)
	buildIndex = func(n *html.Node) {
		if n.Type == html.TextNode {
			start := offset
			offset += len(n.Data)

			segments = append(segments, TextSegment{
				Node:  n,
				Start: start,
				End:   offset,
			})
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			buildIndex(c)
		}
	}

	buildIndex(doc)

	annotationOverlaps := func(aStart, aEnd, bStart, bEnd int) bool {
		return aStart < bEnd && bStart < aEnd
	}

	for _, seg := range segments {
		var newNodes []*html.Node
		cursor := 0

		for _, a := range annotations {

			if !annotationOverlaps(a.Start, a.End, seg.Start, seg.End) {
				continue
			}

			localStart := max(0, a.Start-seg.Start)
			localEnd := min(len(seg.Node.Data), a.End-seg.Start)

			// before
			if cursor < localStart {
				newNodes = append(newNodes, textNode(seg.Node.Data[cursor:localStart]))
			}

			span := &html.Node{
				Type: html.ElementNode,
				Data: "span",
				Attr: []html.Attribute{
					{Key: "class", Val: "highlight"},
					{Key: "data-id", Val: a.ID},
				},
			}

			// operations
			span.AppendChild(textNode(seg.Node.Data[localStart:localEnd]))
			newNodes = append(newNodes, span)

			cursor = localEnd
		}

		// remainder
		if cursor < len(seg.Node.Data) {
			newNodes = append(newNodes, textNode(seg.Node.Data[cursor:]))
		}

		// replace original node
		if len(newNodes) > 0 {
			parent := seg.Node.Parent
			for _, n := range newNodes {
				parent.InsertBefore(n, seg.Node)
			}
			parent.RemoveChild(seg.Node)
		}
	}

	var buf strings.Builder
	html.Render(&buf, doc)
	return buf.String(), nil
}

func textNode(s string) *html.Node {
	return &html.Node{
		Type: html.TextNode,
		Data: s,
	}
}
