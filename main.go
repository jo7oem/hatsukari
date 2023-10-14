package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"time"
)

func main() {
	db, err := sql.Open("postgres", "user=hatsukari dbname=hatsukari password=hatsukari sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	fmt.Print("Wake up!")
	for {
		fmt.Println("work!")
		log.Println("error work!")
		time.Sleep(1 * time.Second)
	}
}
