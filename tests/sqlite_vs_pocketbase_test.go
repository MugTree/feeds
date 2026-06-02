package tests

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

var (
	db            *sql.DB
	httpClient    = &http.Client{Timeout: 5 * time.Second}
	pocketBaseURL = "http://localhost:8090/api/collections/cars/records?filter=(url='some-url')"
)

func BenchmarkSQLiteFetch(b *testing.B) {
	var err error
	db, err = sql.Open("sqlite", "../data/feedreader.db")
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	for b.Loop() {
		var id int
		var name string
		err := db.QueryRow("SELECT id, url FROM feeds WHERE id = ?", 1).Scan(&id, &name)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPocketBaseFetch(b *testing.B) {

	for b.Loop() {
		resp, err := httpClient.Get(pocketBaseURL) // Fetch record with ID=1
		if err != nil {
			b.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			b.Fatal(err)
		}

		var data map[string]any
		err = json.Unmarshal(body, &data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
