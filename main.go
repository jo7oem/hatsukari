package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/jo7oem/hatsukari/store/config"
	"github.com/jo7oem/hatsukari/store/rdb"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/jo7oem/hatsukari/store/contents"
	_ "github.com/lib/pq"           //nolint:depguard
	_ "github.com/mattn/go-sqlite3" //nolint:depguard
)

const (
	DB_POSTGRES = "postgres"
	DB_SQLITE   = "sqlite3"
)

var (
	ErrLocked    = fmt.Errorf("already locked")
	ErrUnLocked  = fmt.Errorf("already unlocked")
	ErrNilConfig = fmt.Errorf("no config")
)

//go:embed db/migrations/*.sql
var migrateFiles embed.FS

func main() {
	var db *sql.DB

	conf, err := config.LoadConfig("config_my.yml")
	if err != nil {
		panic(err)
	}

	var driver database.Driver

	switch conf.DB.DBType {
	case DB_POSTGRES:
		tdb, err := sql.Open("postgres", "user=hatsukari dbname=hatsukari password=hatsukari sslmode=disable")
		if err != nil {
			panic(err)
		}

		db = tdb
		defer db.Close()

		driver, err = postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			panic(err)
		}
	case DB_SQLITE:
		tdb, err := sql.Open("sqlite3", "hatsukari.db")
		if err != nil {
			panic(err)
		}

		db = tdb

		defer db.Close()

		driver, err = sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			panic(err)
		}
	}

	if err := rdb.MigrateDB(driver, migrateFiles); err != nil {
		panic(err)
	}

	if !conf.Contents.IsPathExist() {
		if conf.Contents.CreatePath() != nil {
			panic("failed to create path")
		}

		fmt.Println("path created")
	}

	if err := conf.Contents.Load(); err != nil {
		panic(err)
	}

	fmt.Print("Wake up!") //nolint:forbidigo

	mux := http.NewServeMux()
	mux.HandleFunc("/", contents.BlogRouter)
	mux.HandleFunc("/static/", contents.StaticRouter)
	srv := http.Server{
		Addr:                         "localhost:8080",
		Handler:                      mux,
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    nil,
		ReadTimeout:                  0,
		ReadHeaderTimeout:            0,
		WriteTimeout:                 0,
		IdleTimeout:                  0,
		MaxHeaderBytes:               0,
		TLSNextProto:                 nil,
		ConnState:                    nil,
		ErrorLog:                     nil,
		BaseContext:                  nil,
		ConnContext:                  nil,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}

		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed

	fmt.Println("shutdown")
}
