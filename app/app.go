package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mugtree/feeds/app/db"
)

type App struct {
	dbx     *sqlx.DB
	db      *sql.DB
	queries *db.Queries
	server  *http.Server
	ctx     context.Context
	cancel  context.CancelFunc
	errCh   chan error

	wg       sync.WaitGroup
	stopOnce sync.Once
}

func NewApp(db *sql.DB, queries *db.Queries, server *http.Server) *App {

	ctx, cancel := context.WithCancel(context.Background())

	dbx := sqlx.NewDb(db, "sqlite3")

	return &App{
		dbx:     dbx,
		db:      db, // lifescyle only
		queries: queries,
		server:  server,
		ctx:     ctx,
		cancel:  cancel,
		errCh:   make(chan error, 1),
	}
}

func (a *App) Start() {

	// HTTP
	a.wg.Go(func() {

		LogInfo("http server starting")

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.errCh <- err
		}
	})

	// Add anything else you need here

}
func (a *App) Stop() {
	a.stopOnce.Do(func() {

		LogInfo("shutdown initiated")

		// this will effectively stop the two routines that start up on the Start method
		// as they select for  ctx.Done()
		a.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := a.server.Shutdown(ctx); err != nil {
			LogError("http shutdown error: " + err.Error())
		}

		a.wg.Wait()

		if err := a.db.Close(); err != nil {
			LogError("db close error: " + err.Error())
		}
	})
}

func (a *App) Wait() error {
	done := make(chan struct{})

	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case err := <-a.errCh:

		a.Stop()

		<-done

		return err

	case <-done:
		return nil
	}
}

func LogInfo(msg string) {
	log.Println("INFO: " + msg)
}

func LogError(msg string) {
	log.Println("ERROR: " + msg)
}
