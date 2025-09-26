package main

import (
	"asset/database"
	"asset/internal"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	db     *mongo.Database
	client *mongo.Client
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//initializing db
	var err error
	db, client, err = database.NewDB(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer client.Disconnect(ctx)

	fs := http.FileServer(http.Dir("style"))

	//initialising router
	r := mux.NewRouter()
	r.PathPrefix("/style/").Handler(http.StripPrefix("/style/", fs))
	r.HandleFunc("/assets", internal.GetAssets(db)).Methods("GET")
	r.HandleFunc("/assets", internal.AddAsset(db)).Methods("POST")
	r.HandleFunc("/assets/{id}", internal.GetAsset(db)).Methods("GET")
	r.HandleFunc("/assets/{id}/edit", internal.EditAsset(db)).Methods("POST")
	r.HandleFunc("/assets/{id}/delete", internal.DeleteAsset(db)).Methods("POST")



	fmt.Printf("Using database: %v", db.Name())

	//Intialising server
	http.ListenAndServe("localhost:5500", r)
}
