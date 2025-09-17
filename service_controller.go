package main

import (
	"context"
	"html/template"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func serviceListHandler(w http.ResponseWriter, r *http.Request) {
	cur, err := serviceCollection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to retrieve services", http.StatusInternalServerError)
		return
	}
	var services []Service
	cur.All(context.Background(), &services)

	data := struct {
		Services []Service
		Error    string
	}{
		Services: services,
		Error:    "",
	}

	tmpl := template.Must(template.ParseFiles("templates/service.html"))
	tmpl.Execute(w, data)
}

// Create Service
func serviceCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		label := r.FormValue("label")
		notes := r.FormValue("notes")

		if label == "" {
			cur, err := serviceCollection.Find(context.Background(), bson.M{})
			if err != nil {
				http.Error(w, "Failed to retrieve services", http.StatusInternalServerError)
				return
			}
			var services []Service
			cur.All(context.Background(), &services)

			data := struct {
				Services []Service
				Error    string
			}{
				Services: services,
				Error:    "Label is required!",
			}
			tmpl := template.Must(template.ParseFiles("templates/service.html"))
			tmpl.Execute(w, data)
			return
		}

		serviceCollection.InsertOne(context.Background(), Service{
			ID:    primitive.NewObjectID(),
			Label: label,
			Notes: notes,
		})
		http.Redirect(w, r, "/service", http.StatusSeeOther)
	}
}

func serviceEditHandler(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodPost {
		label := r.FormValue("label")
		notes := r.FormValue("notes")
		if label == "" {
			data := struct{ Error string }{Error: "Label is required!"}
			tmpl := template.Must(template.ParseFiles("templates/service.html"))
			tmpl.Execute(w, data)
			return
		}
		serviceCollection.UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": bson.M{"label": label, "notes": notes}})
		http.Redirect(w, r, "/service", http.StatusSeeOther)
	}
}

func serviceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	serviceCollection.DeleteOne(context.Background(), bson.M{"_id": id})
	http.Redirect(w, r, "/service", http.StatusSeeOther)
}
