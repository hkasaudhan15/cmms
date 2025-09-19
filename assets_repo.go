package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getAllAssets(ctx context.Context, db *mongo.Database) ([]Asset, error) {
	var result []Asset
	assetCollection := db.Collection("assets")

	cur, err := assetCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	err = cur.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func insertAsset(ctx context.Context, db *mongo.Database, asset Asset) error {
	collection := db.Collection("assets")
	_, err := collection.InsertOne(ctx, asset)
	return err
}