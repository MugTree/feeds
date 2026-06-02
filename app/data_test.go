package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

/*
This test checks that the different strategies we use for clipping a piece of html out of a 'container'(ie <div id="all"></div> )match up with expectations - There are four different strategies (all, clipStart, clipEnd, clipBetween)
*/
func Test_ExtractHTMLRangeFlat(t *testing.T) {

	var htmlString = `<!doctype html><html lang="en"><head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><title>Document</title></head><body><div id="all"><h1>Heading</h1><p> Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. </p><p> Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. </p><p> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. </p><p> Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. </p><div id="nasty-ads">Some nasty ads</div></div></body></html>`

	var clippedStartString = `<p> Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. </p><p> Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. </p><p> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. </p><p> Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. </p><div id="nasty-ads">Some nasty ads</div>`

	var clippedEndString = `<h1>Heading</h1><p> Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. </p><p> Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. </p><p> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. </p><p> Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. </p>`

	var unclippedString = `<h1>Heading</h1><p> Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. </p><p> Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. </p><p> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. </p><p> Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. </p><div id="nasty-ads">Some nasty ads</div>`

	var clippedBetweenString = `<p> Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. </p><p> Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. </p><p> Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. </p><p> Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. </p>`

	t.Run("test html extraction", func(t *testing.T) {

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(htmlString))
		}))
		defer ts.Close()

		var result string

		c := colly.NewCollector()

		var theDom *goquery.Selection

		c.OnHTML("#all", func(h *colly.HTMLElement) {
			theDom = h.DOM
			result = ExtractHTMLRangeFlat(theDom, "p:first-of-type", "")
		})

		if err := c.Visit(ts.URL); err != nil {
			t.Fatalf("visit failed: %v", err)
		}

		// return clipped from the first paragraph
		//----------------------------------------
		expected := clippedStartString
		if result != expected {
			t.Errorf("Expected:\n'%s'\nGot:\n'%s", expected, result)
		}

		// return unclipped
		//----------------------------------------
		result = ExtractHTMLRangeFlat(theDom, "", "")

		expected = unclippedString

		if result != expected {
			t.Errorf("Expected:\n'%s'\nGot:\n'%s", expected, result)
		}

		// return with the end clipped
		//----------------------------------------
		result = ExtractHTMLRangeFlat(theDom, "", "#nasty-ads")

		expected = clippedEndString

		if result != expected {
			t.Errorf("Expected:\n'%s'\nGot:\n'%s", expected, result)
		}

		// return  clipped between
		//----------------------------------------
		result = ExtractHTMLRangeFlat(theDom, "p:first-of-type", "#nasty-ads")

		expected = clippedBetweenString

		if result != expected {
			t.Errorf("Expected:\n'%s'\nGot:\n'%s", expected, result)
		}

	})

}
