package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mugtree/feeds/app/db"
)

type App struct {
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

	return &App{
		db:      db, // lifescyle only - shutdown etc
		queries: queries,
		server:  server,
		ctx:     ctx,
		cancel:  cancel,
		errCh:   make(chan error, 1),
	}
}

func (a *App) Start() {

	a.wg.Go(func() {

		LogInfo("http server starting")

		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.errCh <- err
		}
	})

	/**
	-------------------------
	Thinking here is that if any other concurrent processes need running I can add them here
	Thinks like... data cleanup and deletion. Auto loading feeds if not updated daily etc.

	a.wg.Go(func() {
		LogInfo("starting other process")
		...
	})
	-------------------------
	*/
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
