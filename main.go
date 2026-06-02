package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mugtree/feeds/app"
	"github.com/mugtree/feeds/app/db"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if err := run(ctx); err != nil {
		app.LogError(err.Error())
	}
}

func run(parent context.Context) error {

	mustEnv := func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok {
			log.Fatalf("missing .env: %s", key)
		}
		return val
	}

	appPort := mustEnv("APP_PORT")
	appDb := mustEnv("APP_DB")
	appUser := mustEnv("APP_USER")
	appPassword := mustEnv("APP_PASSWORD")

	dbHandle, err := sql.Open("sqlite3", appDb)
	if err != nil {
		return err
	}

	_, _ = dbHandle.Exec(`PRAGMA journal_mode=WAL;`)

	if err := dbHandle.Ping(); err != nil {
		return err
	}

	queries := db.New(dbHandle)

	webserver := &http.Server{
		Addr:    ":" + appPort,
		Handler: app.SetupHttpServer(queries, appUser, appPassword),
	}

	application := app.NewApp(dbHandle, queries, webserver)

	// bind OS signal context → app shutdown
	go func() {
		<-parent.Done()
		application.Stop()
	}()

	application.Start()

	if err := application.Wait(); err != nil {
		return err
	}

	app.LogInfo("server stopped cleanly")
	return nil

}
