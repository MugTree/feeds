package main

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

/* Writing some "tests" to get a better grip of the AI written code */

func main() {

	htmlStr := `<main id="base" onmouseup="data = getSelectionRangeData(); console.log(data)"><p>ABC</p><ul><li>DEF</li><li>GHI</li><li>JKL</li></ul><p>MNO</p><table><tbody><tr><th>abc</th></tr><tr><td>123</td></tr></tbody></table></main>`

	a := NewAnnotation{
		ID: "adsfasd",
		Start: Position{
			Path:   []int{0, 0},
			Offset: 0,
		},
		End: Position{
			Path:   []int{3, 0, 0},
			Offset: 2,
		},
	}

	ans := []NewAnnotation{a}

	output, err := ApplyAnnotations(htmlStr, ans)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(output)

}

type Position struct {
	Path   []int `json:"path"`
	Offset int   `json:"offset"`
}

type NewAnnotation struct {
	ID    string   `json:"id"`
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type TextNode struct {
	Node  *html.Node
	Path  string
	Index int
	Start int
	End   int
}

type TextIndex struct {
	Nodes  []*TextNode
	Lookup map[string]*TextNode
}

type ResolvedAnnotation struct {
	ID string

	StartNode *TextNode
	EndNode   *TextNode

	StartOffset int
	EndOffset   int
}

func findNode(n *html.Node, tag string) *html.Node {

	var walk func(*html.Node) *html.Node

	walk = func(n *html.Node) *html.Node {

		if n.Type == html.ElementNode && n.Data == tag {
			return n
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if res := walk(c); res != nil {
				return res
			}
		}

		return nil
	}

	return walk(n)
}

func ApplyAnnotations(htmlInput string, annotations []NewAnnotation) (string, error) {

	doc, err := html.Parse(strings.NewReader(htmlInput))
	if err != nil {
		return "", err
	}

	article := findNode(doc, "main")

	index := buildTextIndex(article)

	resolved, err := resolveAnnotations(index, annotations)
	if err != nil {
		return "", err
	}

	renderAnnotations(index, resolved)

	var buf strings.Builder
	html.Render(&buf, doc)

	return buf.String(), nil
}

func buildTextIndex(root *html.Node) *TextIndex {

	index := &TextIndex{
		Nodes:  make([]*TextNode, 0),
		Lookup: make(map[string]*TextNode),
	}

	offset := 0

	var walk func(*html.Node, []int)

	walk = func(n *html.Node, path []int) {

		if n.Type == html.TextNode {

			key := pathToString(path)

			tn := &TextNode{
				Node:  n,
				Path:  key,
				Index: len(index.Nodes),
				Start: offset,
				End:   offset + len(n.Data),
			}

			index.Nodes = append(index.Nodes, tn)
			index.Lookup[key] = tn

			offset += len(n.Data)
		}

		i := 0
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, append(path, i))
			i++
		}
	}

	walk(root, nil)

	return index
}

func resolveAnnotations(index *TextIndex, annotations []NewAnnotation) ([]ResolvedAnnotation, error) {

	var resolved []ResolvedAnnotation

	for _, a := range annotations {

		startNode, startOffset, err := resolvePosition(index, a.Start)
		if err != nil {
			return nil, err
		}

		endNode, endOffset, err := resolvePosition(index, a.End)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, ResolvedAnnotation{
			ID: a.ID,

			StartNode: startNode,
			EndNode:   endNode,

			StartOffset: startOffset,
			EndOffset:   endOffset,
		})
	}

	return resolved, nil
}

func resolvePosition(index *TextIndex, pos Position) (*TextNode, int, error) {

	key := pathToString(pos.Path)

	node, ok := index.Lookup[key]
	if !ok {
		return nil, 0, fmt.Errorf("text node not found: %s", key)
	}

	return node, pos.Offset, nil
}

func renderAnnotations(index *TextIndex, annotations []ResolvedAnnotation) {

	for _, t := range index.Nodes {

		var newNodes []*html.Node
		cursor := 0

		for _, a := range annotations {

			if !overlaps(a, t) {
				continue
			}

			spanStart, spanEnd := localOffsets(a, t)

			if cursor < spanStart {
				newNodes = append(newNodes,
					textNode(t.Node.Data[cursor:spanStart]))
			}

			span := &html.Node{
				Type: html.ElementNode,
				Data: "span",
				Attr: []html.Attribute{
					{Key: "class", Val: "highlight"},
					{Key: "data-id", Val: a.ID},
				},
			}

			span.AppendChild(
				textNode(t.Node.Data[spanStart:spanEnd]),
			)

			newNodes = append(newNodes, span)

			cursor = spanEnd
		}

		if cursor < len(t.Node.Data) {
			newNodes = append(newNodes,
				textNode(t.Node.Data[cursor:]))
		}

		if len(newNodes) == 0 {
			continue
		}

		parent := t.Node.Parent

		for _, n := range newNodes {
			parent.InsertBefore(n, t.Node)
		}

		parent.RemoveChild(t.Node)
	}
}

func overlaps(a ResolvedAnnotation, t *TextNode) bool {

	return t.Index >= a.StartNode.Index &&
		t.Index <= a.EndNode.Index
}

func localOffsets(a ResolvedAnnotation, t *TextNode) (int, int) {

	start := 0
	end := len(t.Node.Data)

	if t == a.StartNode {
		start = a.StartOffset
	}

	if t == a.EndNode {
		end = a.EndOffset
	}

	return start, end
}

func textNode(s string) *html.Node {

	return &html.Node{
		Type: html.TextNode,
		Data: s,
	}
}

func pathToString(path []int) string {

	var b strings.Builder

	for i, p := range path {

		if i > 0 {
			b.WriteByte('.')
		}

		b.WriteString(strconv.Itoa(p))
	}

	return b.String()
}
