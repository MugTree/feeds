package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mmcdole/gofeed"
	"github.com/mugtree/feeds/app/db"
)

/*

This file contains functions that reach out to the net to do stuff

*/

func feedsNetRetrieveAndSanitizeArticleHTML(_ *db.Queries, afd db.SelectFeedAndArticletByArticleIDRow, _ context.Context) (string, int64, error) {

	pageHtmlContent := ""

	type extractionParams struct {
		Container      string
		ClipStartPoint string
		ClipEndPoint   string
	}

	ep := extractionParams{}
	ep.Container = afd.FeedCssSelContainer.String

	switch afd.FeedHtmlExtractionStrategy.String {
	case "no-clip":
		break
	case "clip-start":
		ep.ClipStartPoint = afd.FeedCssSelStart.String
	case "clip-end":
		ep.ClipEndPoint = afd.FeedCssSelStop.String
	case "clip-between":
		ep.ClipStartPoint = afd.FeedCssSelStart.String
		ep.ClipEndPoint = afd.FeedCssSelStop.String
	}

	//TODO - need to add some timeout values here really
	c := colly.NewCollector()

	c.OnHTML(ep.Container, func(h *colly.HTMLElement) {
		pageHtmlContent = feedsExtractHTMLRangeFlat(h.DOM, ep.ClipStartPoint, ep.ClipEndPoint)
	})

	if err := c.Visit(afd.ArticleLink); err != nil {
		return "", 0, fmt.Errorf("error using colly to visit page: %v - %v", afd.ArticleLink, err)
	}

	sanitizedHtml, clickableBlockCount, err := feedsSanitizeAndAnnotateHTMLForStorage(pageHtmlContent)
	if err != nil {
		return "", 0, err
	}
	fmt.Println("feedsGetArticleHTMLFromWeb")
	fmt.Printf("Counted %v clickable blocks...\n\n", clickableBlockCount)

	stringifiedHTML, err := feedsStringifyHTML(sanitizedHtml)
	if err != nil {
		return "", 0, err
	}

	return stringifiedHTML, clickableBlockCount, nil
}

func feedsNetGetFeedUpdates(queries *db.Queries, ctx context.Context) (int64, error) {

	feeds, err := queries.SelectAllFeeds(ctx)
	if err != nil {
		return 0, fmt.Errorf("get feeds: %w", err)
	}

	parser := gofeed.NewParser()
	parser.Client = &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, feed := range feeds {

		goFeed, err := parser.ParseURL(fmt.Sprintf("%s/feed/", feed.Url))
		if err != nil {
			return 0, fmt.Errorf("parse feed %s: %w", feed.Url, err)
		}

		if goFeed == nil {
			continue
		}

		for _, item := range goFeed.Items {

			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
			}

			now := time.Now()

			sanitizedHtml, clickableBlockCount, err := feedsSanitizeAndAnnotateHTMLForStorage(item.Description)
			if err != nil {
				return 0, err
			}

			fmt.Println("feedsGetFeedUpdates")
			fmt.Printf("Counted %v blocks...\n\n", clickableBlockCount)

			output, err := feedsStringifyHTML(sanitizedHtml)
			if err != nil {
				return 0, err
			}

			err = queries.InsertOrIgnoreArticle(ctx, db.InsertOrIgnoreArticleParams{
				FeedID:    feed.ID,
				Title:     item.Title,
				Link:      item.Link,
				Published: feedsGetFeedItemDate(item),
				DateFound: &now,
				Summary:   output,
				Read:      0,
				Starred:   0,
			})
			if err != nil {
				return 0, fmt.Errorf("insert article: %w", err)
			}
		}
	}

	return int64(len(feeds)), nil
}
