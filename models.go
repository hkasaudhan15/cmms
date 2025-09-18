package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

type Shedule struct {
	ID          primitive.ObjectID   `bson:"_id"`
	Lable       string               `bson:"label"`
	SheduleType string               `bson:"shedule_type"`
	Days        int                  `bson:"days"`
	Services    []primitive.ObjectID `bson:"services"`
	Consumables []primitive.ObjectID `bson:"consumables"`
	Notes       string               `bson:"notes"`
}

type MainteneceShedule struct {
	ID       primitive.ObjectID `bson:"_id"`
	Lable    string             `bson:"label"`
	AssetID  primitive.ObjectID `bson:"asset_id"`
	Shedules []Shedule          `bson:"shedules"`
}

type Asset struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Label         string             `bson:"label" json:"label"`
	Type          string             `bson:"type" json:"type"`
	Location      string             `bson:"location" json:"location"`
	EffectiveDate time.Time          `bson:"effective_date" json:"effective_date"`
}

type AllAssets struct {
	Data    []Asset
	Error   string
}