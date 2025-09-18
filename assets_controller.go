package main

import (
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

func getAssets(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var result AllAssets

		data, err := getAllAssets(ctx, db)
		if err != nil {
			log.Printf("error fetching records: %v", err)
			result.Error = "error fetching records"

		} else {
			result.Data = data
		}

		templates.ExecuteTemplate(w, "Login.html",result)
	}
}
