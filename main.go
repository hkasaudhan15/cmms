package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
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

	fmt.Printf("Using database: %v", db.Name())

	//setting up router and routes
	router := mux.NewRouter()


	//Intialising server
	fmt.Println("server running on port 5500")
	if err := http.ListenAndServe(":5500", router); err != nil {
		log.Fatal(err)
	}
}
