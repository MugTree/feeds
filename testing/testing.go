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

	c := strings.NewReader(htmlStr)

	root, err := html.Parse(c)
	if err != nil {
		fmt.Println(err)
		return
	}

	index := buildTextIndex(root)
	fmt.Printf("length: %v", len(index.Nodes))
}

type Position struct {
	Path   []int `json:"path"`
	Offset int   `json:"offset"`
}
type New_Annotation struct {
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

// type ResolvedAnnotation struct {
// 	ID string

// 	StartNode *TextNode
// 	EndNode   *TextNode

// 	StartOffset int
// 	EndOffset   int
// }

// func ApplyAnnotations(htmlInput string, annotations []New_Annotation) (string, error) {
// 	return "", nil
// }

func buildTextIndex(root *html.Node) *TextIndex {

	index := &TextIndex{
		Nodes:  make([]*TextNode, 0),
		Lookup: make(map[string]*TextNode),
	}

	// will be incremented based on length of text content
	offset := 0

	var walk func(*html.Node, []int)
	walk = func(n *html.Node, path []int) {

		if n.Type == html.TextNode {
			key := pathToString(path)

			tn := &TextNode{
				Node:  n,
				Path:  key,
				Index: len(index.Nodes), // length of TextNode slice
				Start: offset,
				End:   offset + len(n.Data),
			}

			index.Nodes = append(index.Nodes, tn)
			index.Lookup[key] = tn

			offset += len(n.Data) //offset based on length of text content

		}

		i := 0
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, append(path, i))
			i++
		}

	}

	walk(root, []int{})

	return index

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
