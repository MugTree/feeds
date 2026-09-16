package main

import "github.com/goforj/godump"

func main() {

	// a page can have 3 paragraphs how many pages do we need

	// numberOfParagraphs := 13

	// if numberOfParagraphs == 0 {
	// 	return
	// }

	// pagesRequired := 1

	// for i := range numberOfParagraphs {

	// 	if (i+1)%3 == 0 {
	// 		pagesRequired++
	// 	}
	// }

	// godump.Dump("pagesRequired: ", pagesRequired)

	fruit := []string{"pears", "apples", "oranges", "grapes"}

	last := fruit[len(fruit)-1]
	rest := fruit[:len(fruit)-1]

	godump.Dump("last", last, "rest", rest)

}
