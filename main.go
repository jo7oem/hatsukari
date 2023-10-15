package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq" //nolint:depguard
)

func main() {
	db, err := sql.Open("postgres", "user=hatsukari dbname=hatsukari password=hatsukari sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Print("Wake up!") //nolint:forbidigo

	for {
		fmt.Println("work!") //nolint:forbidigo
		log.Println("error work!")
		time.Sleep(1 * time.Second)
	}
}
