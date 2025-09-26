package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	db                   *mongo.Database
	client               *mongo.Client
	templates            *template.Template
	consumableCollection *mongo.Collection
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error
	db, client, err = NewDB(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer client.Disconnect(ctx)

	consumableCollection = db.Collection("consumables")

	templates = template.Must(template.ParseGlob("templates/*.html"))

	fs := http.FileServer(http.Dir("style"))
	http.Handle("/style/", http.StripPrefix("/style/", fs))

	// Routes
	http.HandleFunc("/consumable", consumableListHandler)
	http.HandleFunc("/consumable/create", consumableCreateHandler)
	http.HandleFunc("/consumable/edit", consumableEditHandler)
	http.HandleFunc("/consumable/delete", consumableDeleteHandler)


	fmt.Println("Consumable microservice running on :8082")
	http.ListenAndServe("localhost:8082", nil)
}
