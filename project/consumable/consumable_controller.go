package main

import (
	"context"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// List Consumables
func consumableListHandler(w http.ResponseWriter, r *http.Request) {
	cur, err := consumableCollection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to retrieve consumables", http.StatusInternalServerError)
		return
	}
	var consumables []Consumable
	cur.All(context.Background(), &consumables)

	data := struct {
		Consumables []Consumable
		Error       string
	}{
		Consumables: consumables,
		Error:       "",
	}

	templates.ExecuteTemplate(w, "consumable.html", data)
}

// Create Consumable
func consumableCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		label := r.FormValue("label")
		notes := r.FormValue("notes")

		if label == "" {
			cur, _ := consumableCollection.Find(context.Background(), bson.M{})
			var consumables []Consumable
			cur.All(context.Background(), &consumables)

			data := struct {
				Consumables []Consumable
				Error       string
			}{
				Consumables: consumables,
				Error:       "Label is required!",
			}
			templates.ExecuteTemplate(w, "consumable.html", data)
			return
		}

		consumableCollection.InsertOne(context.Background(), Consumable{
			ID:    primitive.NewObjectID(),
			Label: label,
			Notes: notes,
		})
		http.Redirect(w, r, "/consumable", http.StatusSeeOther)
	}
}

// Edit Consumable
func consumableEditHandler(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		label := r.FormValue("label")
		notes := r.FormValue("notes")

		if label == "" {
			cur, _ := consumableCollection.Find(context.Background(), bson.M{})
			var consumables []Consumable
			cur.All(context.Background(), &consumables)

			data := struct {
				Consumables []Consumable
				Error       string
			}{
				Consumables: consumables,
				Error:       "Label is required!",
			}
			templates.ExecuteTemplate(w, "consumable.html", data)
			return
		}

		consumableCollection.UpdateOne(context.Background(),
			bson.M{"_id": id},
			bson.M{"$set": bson.M{"label": label, "notes": notes}})
		http.Redirect(w, r, "/consumable", http.StatusSeeOther)
	}
}

// Delete Consumable
func consumableDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	consumableCollection.DeleteOne(context.Background(), bson.M{"_id": id})
	http.Redirect(w, r, "/consumable", http.StatusSeeOther)
}
