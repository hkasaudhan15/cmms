package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//initializing db
	db, _, err := NewDB(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}

	fs := http.FileServer(http.Dir("style"))
	http.Handle("/style/", http.StripPrefix("/style/", fs))

	http.HandleFunc("/services", serviceHandler)

	fmt.Printf("Using database: %v", db.Name())

	//Intialising server
	http.ListenAndServe(":8080", nil)
}
