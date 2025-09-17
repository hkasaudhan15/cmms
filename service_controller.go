package main

import (
	"context"
	"html/template"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
)

func serviceHandler(w http.ResponseWriter, r *http.Request) {
	code, err := serviceCollection.Find(context.Background(), bson.M{})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var services []Service
	code.All(context.Background(), &services)
	template := template.Must(template.ParseFiles("templates/service.html"))
	template.Execute(w, services)

}
