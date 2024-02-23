package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"           //nolint:depguard
	_ "github.com/mattn/go-sqlite3" //nolint:depguard
)

const (
	DB_POSTGRES = "postgres"
	DB_SQLITE   = "sqlite3"
)

var useDB = DB_SQLITE

var (
	ErrLocked    = fmt.Errorf("already locked")
	ErrUnLocked  = fmt.Errorf("already unlocked")
	ErrNilConfig = fmt.Errorf("no config")
)

//go:embed db/migrations/*.sql
var migrateFiles embed.FS

func main() {
	var db *sql.DB

	var driver database.Driver

	switch useDB {
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

	fmt.Print("Wake up!") //nolint:forbidigo

	for {
		fmt.Println("work!") //nolint:forbidigo
		log.Println("error work!")
		time.Sleep(1 * time.Second)
	}
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
