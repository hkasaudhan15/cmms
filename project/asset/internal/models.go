package internal

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Asset struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Label         string             `bson:"label" json:"label"`
	Type          string             `bson:"type" json:"type"`
	Location      string             `bson:"location" json:"location"`
	EffectiveDate time.Time          `bson:"effective_date" json:"effective_date"`
}

type AssetsPageData struct {
	Data    []Asset
	Message string
	Error   string
}
