package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/mongo/options"
)

var serviceCollection *mongo.Collection
var consumableCollection *mongo.Collection
var schedulesCollection *mongo.Collection

func NewDB(ctx context.Context) (*mongo.Database, *mongo.Client, error) {
	mongoURI := "mongodb://localhost:27017"
	dbName := "CMMS"

	clientOptions := options.Client().ApplyURI(mongoURI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating server connection: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("error pinging server: %v", err)
	}

	db := client.Database(dbName)
	serviceCollection = db.Collection("services")
	consumableCollection = db.Collection("consumables")
	schedulesCollection = db.Collection("schedules")

	log.Println("successfully connected to the database")

	return db, client, nil
}
