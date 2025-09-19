package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func updateAsset(ctx context.Context, db *mongo.Database, id primitive.ObjectID, asset Asset) error {
	_, err := db.Collection("assets").UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"label":          asset.Label,
			"type":           asset.Type,
			"location":       asset.Location,
			"effective_date": asset.EffectiveDate,
		}},
	)
	return err
}

func deleteAssetByID(ctx context.Context, db *mongo.Database, id primitive.ObjectID) error {
	_, err := db.Collection("assets").DeleteOne(ctx, bson.M{"_id": id})
	return err
}