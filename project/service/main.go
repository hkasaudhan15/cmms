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
	db                *mongo.Database
	client            *mongo.Client
	templates         *template.Template
	serviceCollection *mongo.Collection
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize MongoDB
	var err error
	db, client, err = NewDB(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer client.Disconnect(ctx)

	serviceCollection = db.Collection("services")

	templates = template.Must(template.ParseGlob("templates/*.html"))

	fs := http.FileServer(http.Dir("style"))
	http.Handle("/style/", http.StripPrefix("/style/", fs))

	// Service routes
	http.HandleFunc("/service", serviceListHandler)
	http.HandleFunc("/service/create", serviceCreateHandler)
	http.HandleFunc("/service/edit", serviceEditHandler)
	http.HandleFunc("/service/delete", serviceDeleteHandler)

	// API routes for other microservices
	http.HandleFunc("/services", serviceAPIHandler)

	fmt.Println("Service microservice running on :8081")
	http.ListenAndServe("localhost:8081", nil)
}
