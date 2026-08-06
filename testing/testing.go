package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mugtree/feeds/app/db"
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

	err = queries.UpdateMarginNoteByArticleIDAndBlockID(context.Background(),
		db.UpdateMarginNoteByArticleIDAndBlockIDParams{
			Note:      "update 2",
			ArticleID: 31,
			BlockID:   0,
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

}
