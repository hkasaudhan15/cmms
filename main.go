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
	db        *mongo.Database
	client    *mongo.Client
	templates *template.Template
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//initializing db
	var err error
	db, client, err = NewDB(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer client.Disconnect(ctx)

	templates = template.Must(template.New("").Funcs(template.FuncMap{
    "add": func(a, b int) int { return a + b },
	}).ParseGlob("templates/*.html"))

	fs := http.FileServer(http.Dir("style"))
	http.Handle("/style/", http.StripPrefix("/style/", fs))

	http.HandleFunc("/service", serviceListHandler)
	http.HandleFunc("/service/create", serviceCreateHandler)
	http.HandleFunc("/service/edit", serviceEditHandler)
	http.HandleFunc("/service/delete", serviceDeleteHandler)

	http.HandleFunc("/consumable", consumableListHandler)
	http.HandleFunc("/consumable/create", consumableCreateHandler)
	http.HandleFunc("/consumable/edit", consumableEditHandler)
	http.HandleFunc("/consumable/delete", consumableDeleteHandler)

	// Maintenance Routes
	http.HandleFunc("/maintenances", listMaintenance)
	http.HandleFunc("/maintenances/create", createMaintenance)
	http.HandleFunc("/maintenances/edit", editMaintenance)
	http.HandleFunc("/maintenances/view", viewMaintenance)
	http.HandleFunc("/maintenances/delete", deleteMaintenance)
	http.HandleFunc("/shedules/add", addShedule)
	http.HandleFunc("/shedules/delete", deleteShedule)

	http.HandleFunc("/assets", getAssets(db))

	fmt.Printf("Using database: %v", db.Name())

	//Intialising server
	http.ListenAndServe("localhost:8080", nil)
}