package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Print("Wake up!")
	for {
		fmt.Println("work!")
		log.Println("error work!")
		time.Sleep(1 * time.Second)
	}
}
