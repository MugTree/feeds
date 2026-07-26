package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/goforj/godump"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mugtree/feeds/app/db"
	"github.com/mugtree/feeds/lib"
)

func main() {

	mustEnv := func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok {
			log.Fatalf("missing .env: %s", key)
		}
		return val
	}

	appDb := mustEnv("APP_DB")

	dbHandle, err := sql.Open("sqlite3", appDb)
	if err != nil {
		fmt.Println(err)
		return
	}
	dbHandle.SetMaxOpenConns(1)
	dbHandle.SetMaxIdleConns(1)

	_, _ = dbHandle.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`)

	if err := dbHandle.Ping(); err != nil {
		fmt.Println(err)
		return
	}

	queries := db.New(dbHandle)

	feeds, err := queries.SelectAllFeeds(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	filteredA := []db.Feed{}

	for _, v := range feeds {
		if v.Url == "https://someurl.net" {
			filteredA = append(filteredA, v)
		}
	}

	fmt.Println(len(filteredA))

	filteredB := lib.Filter(feeds, func(item db.Feed) bool {
		return item.Url == "https://someurl.net"
	})

	fmt.Println(len(filteredB))

	for i := range feeds {
		feeds[i].Title = feeds[i].Title + "Asdfasdf"
	}

	godump.Dump(feeds[0].Title)

	mapB := slices.Clone(feeds)
	usefulTransforms := lib.Map(mapB, func(f db.Feed) string {
		return strings.ToTitle(f.Title)
	})

	godump.Dump(usefulTransforms)

}
