package main

import (
	"encoding/json"
	"net/http"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Helper function to fetch services from API
func fetchServicesFromAPI() ([]Service, error) {
	resp, err := http.Get("http://localhost:5500/services")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var services []Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, err
	}

	return services, nil
}

// Helper function to fetch consumables from API
func fetchConsumablesFromAPI() ([]Consumable, error) {
	resp, err := http.Get("http://localhost:5500/consumables")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var consumables []Consumable
	if err := json.NewDecoder(resp.Body).Decode(&consumables); err != nil {
		return nil, err
	}

	return consumables, nil
}

// Helper function to fetch asset from API
func fetchAssetFromAPI(assetID string) (*Asset, error) {
	resp, err := http.Get("http://localhost:5500/assets/" + assetID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var asset Asset
	if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		return nil, err
	}

	return &asset, nil
}

// Helper function to fetch services and consumables (updated to use API)
func fetchServicesAndConsumables() ([]struct {
	ID    primitive.ObjectID `bson:"_id"`
	Label string             `bson:"label"`
}, []struct {
	ID    primitive.ObjectID `bson:"_id"`
	Label string             `bson:"label"`
}) {
	// Fetch from API
	services, err := fetchServicesFromAPI()
	if err != nil {
		// Handle error appropriately
		services = []Service{}
	}

	consumables, err := fetchConsumablesFromAPI()
	if err != nil {
		// Handle error appropriately
		consumables = []Consumable{}
	}

	// Convert to the expected structure
	serviceStructs := make([]struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	}, len(services))
	for i, svc := range services {
		serviceStructs[i] = struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}{ID: svc.ID, Label: svc.Label}
	}

	consumableStructs := make([]struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	}, len(consumables))
	for i, cons := range consumables {
		consumableStructs[i] = struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}{ID: cons.ID, Label: cons.Label}
	}

	return serviceStructs, consumableStructs
}