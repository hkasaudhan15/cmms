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
	// Removed the line that was causing panic since we don't have subdirectories
	// templates = template.Must(templates.ParseGlob("templates/*/*.html"))

	fs := http.FileServer(http.Dir("style"))
	http.Handle("/style/", http.StripPrefix("/style/", fs))

	
	http.HandleFunc("/maintenances", listMaintenance)
	http.HandleFunc("/maintenances/create", createMaintenance)
	http.HandleFunc("/maintenances/edit", editMaintenance)
	http.HandleFunc("/maintenances/view", viewMaintenance)
	http.HandleFunc("/maintenances/delete", deleteMaintenance)

	// Schedule Routes
	http.HandleFunc("/schedules", listSchedules)
	http.HandleFunc("/schedules/add", addSchedule)
	http.HandleFunc("/schedules/edit", editSchedule)
	http.HandleFunc("/schedules/delete", deleteSchedule)


	fmt.Printf("Using database: %v", db.Name())

	//Intialising server
	http.ListenAndServe("localhost:8080", nil)
}