package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mmcdole/gofeed"

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

	//godump.Dump(feeds)

	db, err := sqlx.Open("sqlite3", *dbPtr)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	p := gofeed.NewParser()

	for _, fi := range feeds {

		goFeed, err := p.ParseURL(fi.Url)
		if err != nil {
			log.Fatalf("error parsing: %v", err)
		}

		feedSqlRes, err := db.Exec(
			`INSERT INTO feeds (
				url, 
				title, 
				css_sel_container,
				css_sel_start,
				css_sel_stop,
				html_extraction_strategy,
				last_fetched
				) VALUES (
				?, 
				?, 
				?, 
				?, 
				?, 
				?, 
				CURRENT_TIMESTAMP
				);`,
			goFeed.Link,
			goFeed.Title,
			fi.CSSSelectorContainer,
			fi.CSSSelectorStart,
			fi.CSSSelectorStop,
			fi.HTMLExtractionStrategy)

		if err != nil {
			log.Fatalf("error opening the db: %v", err)
		}

		id, err := feedSqlRes.LastInsertId()
		if err != nil {
			log.Fatalf("error getting last insert id: %v", err)
		}

		for _, v := range goFeed.Items {

			publishedDate := feedItemDate(v)
			dateFound := time.Now()

			_, err = db.Exec(`
				INSERT INTO articles (
				feed_id, 
				title, 
				link, 
				published, 
				date_found, 
				summary, 
				read, 
				starred
				) VALUES (
				 ?, 
				 ?, 
				 ?, 
				 ?, 
				 ?, 
				 ?, 
				 ?,
				 ?
				 );`,
				id,
				v.Title,
				v.Link,
				publishedDate,
				dateFound,
				v.Description,
				0,
				0,
			)

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
