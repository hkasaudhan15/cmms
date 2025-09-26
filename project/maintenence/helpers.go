package main

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func getCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func buildNameMaps(ctx context.Context, svcIDs, consIDs []primitive.ObjectID) (map[string]string, map[string]string) {
	// Fetch all services and consumables
	allServices, allConsumables := fetchServicesAndConsumables()

	// Create maps for quick lookup
	serviceMap := make(map[primitive.ObjectID]struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	})
	for _, svc := range allServices {
		serviceMap[svc.ID] = svc
	}

	consumableMap := make(map[primitive.ObjectID]struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	})
	for _, cons := range allConsumables {
		consumableMap[cons.ID] = cons
	}

	// Build name maps for the specific IDs we need
	svcNames := map[string]string{}
	for _, id := range svcIDs {
		if svc, exists := serviceMap[id]; exists {
			svcNames[id.Hex()] = svc.Label
		} else {
			svcNames[id.Hex()] = id.Hex()
		}
	}

	consNames := map[string]string{}
	for _, id := range consIDs {
		if cons, exists := consumableMap[id]; exists {
			consNames[id.Hex()] = cons.Label
		} else {
			consNames[id.Hex()] = id.Hex()
		}
	}

	return svcNames, consNames
}

func collectScheduleIDs(schedules []Shedule) ([]primitive.ObjectID, []primitive.ObjectID) {
	svcIDSet := map[primitive.ObjectID]struct{}{}
	consIDSet := map[primitive.ObjectID]struct{}{}
	for _, s := range schedules {
		for _, sid := range s.Services {
			svcIDSet[sid] = struct{}{}
		}
		for _, cid := range s.Consumables {
			consIDSet[cid] = struct{}{}
		}
	}

	var svcIDs []primitive.ObjectID
	for id := range svcIDSet {
		svcIDs = append(svcIDs, id)
	}
	var consIDs []primitive.ObjectID
	for id := range consIDSet {
		consIDs = append(consIDs, id)
	}

	return svcIDs, consIDs
}

// Helper function to get asset label (updated to use API)
func getAssetLabel(ctx context.Context, assetID primitive.ObjectID) string {
	asset, err := fetchAssetFromAPI(assetID.Hex())
	if err != nil || asset == nil {
		return assetID.Hex()
	}

	if asset.Label != "" {
		return asset.Label
	}

	return assetID.Hex()
}