package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/app/scraper"

	_ "github.com/mattn/go-sqlite3"
)

type Feed struct {
	Url                    string
	CSSSelectorContainer   string
	CSSSelectorStart       string
	CSSSelectorStop        string
	HTMLExtractionStrategy string
}

// call this like
// go run ./app/db/seed/generate.go --urls=./app/db/seed/seed.csv --db=./feeds.db

func main() {

	filePtr := flag.String("urls", "", "the file to get the urls from - needs to be broken over lines")
	dbPtr := flag.String("db", "", "path to the db")

	flag.Parse()
	fmt.Println("urls:", *filePtr)

	f, err := os.Open(*filePtr)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	feeds := []Feed{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")

		fi := Feed{
			Url:                    parts[0],
			CSSSelectorContainer:   parts[1],
			CSSSelectorStart:       parts[2],
			CSSSelectorStop:        parts[3],
			HTMLExtractionStrategy: parts[4],
		}

		feeds = append(feeds, fi)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	dbhandle, err := sql.Open("sqlite3", *dbPtr)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer dbhandle.Close()

	queries := db.New(dbhandle)

	p := gofeed.NewParser()

	ctx := context.Background()

	for _, fi := range feeds {

		goFeed, err := p.ParseURL(fi.Url)
		if err != nil {
			log.Fatalf("error parsing: %v", err)
		}

		insertedFeed, err := queries.InsertFeed(ctx, db.InsertFeedParams{
			Url:                    goFeed.Link,
			Title:                  goFeed.Title,
			CssSelContainer:        fi.CSSSelectorContainer, //fi.CSSSelectorContainer},
			CssSelStart:            fi.CSSSelectorStart,
			CssSelStop:             fi.CSSSelectorStop,
			HtmlExtractionStrategy: fi.HTMLExtractionStrategy,
		})

		if err != nil {
			log.Fatalf("error opening the db: %v", err)
		}

		for _, v := range goFeed.Items {

			publishedDate := feedItemDate(v)
			dateFound := time.Now()

			fmt.Println("getting link: ", v.Link)
			html, err := scraper.ScrapeSiteHTML(scraper.PageScrapeParams{
				Link:           v.Link,
				Container:      insertedFeed.CssSelContainer,
				ClipStartPoint: insertedFeed.CssSelStart,
				ClipEndPoint:   insertedFeed.CssSelStop,
			})
			if err != nil {
				log.Fatalf("error getting site html: %v", err)
			}

			fmt.Println("processing html for: ", v.Link)
			processed, blockCount, err := scraper.ProcessScrapedHTML(html)
			if err != nil {
				log.Fatalf("error getting site html: %v", err)
			}

			fmt.Println("inserting record for: ", v.Link)
			_, err = queries.InsertArticle(ctx, db.InsertArticleParams{
				FeedID:                  insertedFeed.ID,
				Title:                   v.Title,
				Link:                    v.Link,
				Published:               publishedDate,
				DateFound:               &dateFound,
				ClickableParagraphCount: blockCount,
				Summary:                 v.Description,
				ScrapedHtml:             html,
				ArticleContent:          processed,
				Read:                    0,
				Starred:                 0,
			})

			if err != nil {
				log.Fatalf("error inserting article: %v", err)
			}

		}

	}
}

func feedItemDate(item *gofeed.Item) *time.Time {
	if item.PublishedParsed != nil {
		return item.PublishedParsed
	}

	if item.UpdatedParsed != nil {
		return item.UpdatedParsed
	}

	return nil
}
