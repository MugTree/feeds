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

	"github.com/mmcdole/gofeed"

	"github.com/mugtree/feeds/app"
	"github.com/mugtree/feeds/app/db"

	_ "github.com/mattn/go-sqlite3"
)

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

	feeds := []db.Feed{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ",")

		fi := db.Feed{
			Url:                    parts[0],
			CssSelContainer:        parts[1],
			CssSelStart:            parts[2],
			CssSelStop:             parts[3],
			HtmlExtractionStrategy: parts[4],
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

	var articlesInserted = 0

	for _, feed := range feeds {

		goFeed, err := p.ParseURL(feed.Url)
		if err != nil {
			log.Fatalf("error parsing: %v", err)
		}

		_, err = queries.InsertFeed(ctx, db.InsertFeedParams{
			Url:                    goFeed.Link,
			Title:                  goFeed.Title,
			CssSelContainer:        feed.CssSelContainer, //fi.CSSSelectorContainer},
			CssSelStart:            feed.CssSelStart,
			CssSelStop:             feed.CssSelStop,
			HtmlExtractionStrategy: feed.HtmlExtractionStrategy,
		})

		if err != nil {
			log.Fatalf("error opening the db: %v", err)
		}

		for _, feedItem := range goFeed.Items {
			app.MpHTMLProcessingPipeline(queries, ctx, feedItem, feed)
			articlesInserted++
		}
	}

	_, err = queries.InsertAndReturnFeedsCallData(
		ctx,
		db.InsertAndReturnFeedsCallDataParams{
			RunType:         "seed",
			ArticlesCreated: int64(articlesInserted),
		},
	)
	if err != nil {
		log.Fatalf("error running log:  %v", err)
	}

}
