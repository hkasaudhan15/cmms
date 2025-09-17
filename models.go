package main

import "go.mongodb.org/mongo-driver/bson/primitive"

type Service struct {
	Id    primitive.ObjectID `bson:"_id,omitempty"`
	Lable string             `bson:"lable,omitempty"`
	Notes string             `bson:"notes,omitempty"`
}
