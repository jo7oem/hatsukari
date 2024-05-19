package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jo7oem/hatsukari/resources"
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

	conf, err := loadConfig("config.yml")
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

	if err := migrateDB(driver); err != nil {
		panic(err)
	}

	if !conf.Content.IsPathExist() {
		if conf.Content.CreatePath() != nil {
			panic("failed to create path")
		}

		fmt.Println("path created")
	}

	fmt.Print("Wake up!") //nolint:forbidigo

	mux := http.NewServeMux()
	mux.HandleFunc("/", resources.BlogRouter)
	mux.HandleFunc("/static/", resources.StaticRouter)
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

func migrateDB(driver database.Driver) error {
	fSrc, err := iofs.New(migrateFiles, "db/migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", fSrc, "sqlite3", driver)
	if err != nil {
		return err
	}

	v, _, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return m.Up()
	}

	if err != nil {
		return err
	}

	if _, err := fSrc.Next(v); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}

	if err := m.Up(); errors.Is(err, migrate.ErrNoChange) {
		return nil
	} else {
		return err
	}
}
