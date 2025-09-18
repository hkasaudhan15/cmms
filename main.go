package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
)

// Declare these global variables here so they're accessible across files
var (
	db        *mongo.Database
	client    *mongo.Client
	templates *template.Template
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database
	db, client, err := NewDB(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer client.Disconnect(ctx)

	// Set global variables
	setDB(db, client)

	// Initialize templates - include nested template files (e.g. templates/maintenence/*.html)
	templates = template.Must(template.ParseGlob("templates/**/*.html"))

	fmt.Printf("Using database: %v", db.Name())

	//setting up router and routes
	router := mux.NewRouter()

	//Intialising server
	fmt.Println("server running on port 5500")
	if err := http.ListenAndServe(":5500", router); err != nil {
		log.Fatal(err)
	}
}

// SetDB function to set the global database variable
func setDB(database *mongo.Database, c *mongo.Client) {
	db = database
	client = c
}
