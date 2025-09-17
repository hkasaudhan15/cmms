package main

import "go.mongodb.org/mongo-driver/bson/primitive"

type Service struct {
	ID    primitive.ObjectID `bson:"_id"`
	Label string             `bson:"label"`
	Notes string             `bson:"notes"`
}

type Consumable struct {
	ID    primitive.ObjectID `bson:"_id"`
	Label string             `bson:"label"`
	Notes string             `bson:"notes"`
}
