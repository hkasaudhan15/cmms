package main

import (
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// List maintenances for a specific asset
func listMaintenance(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		http.Error(w, "Missing asset_id in query", http.StatusBadRequest)
		return
	}

	objAssetID, err := primitive.ObjectIDFromHex(assetID)
	if err != nil {
		http.Error(w, "Invalid asset_id", http.StatusBadRequest)
		return
	}

	ctx, cancel := getCtx()
	defer cancel()

	cursor, err := db.Collection("maintenances").Find(ctx, bson.M{"asset_id": objAssetID})
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var items []MainteneceShedule
	if err := cursor.All(ctx, &items); err != nil {
		http.Error(w, "Decode error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch available services and consumables from API
	services, consumables := fetchServicesAndConsumables()

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

	svcNames := map[string]string{}
	for _, s := range serviceStructs {
		svcNames[s.ID.Hex()] = s.Label
	}
	consNames := map[string]string{}
	for _, c := range consumableStructs {
		consNames[c.ID.Hex()] = c.Label
	}

	assetLabel := getAssetLabel(ctx, objAssetID)

	// Check for any message to display
	message := r.URL.Query().Get("message")
	messageType := r.URL.Query().Get("type")

	data := struct {
		AssetID    string
		AssetLabel string
		Items      []MainteneceShedule
		Services   []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}
		Consumables []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}
		ServiceNames    map[string]string
		ConsumableNames map[string]string
		Message         string
		MessageType     string
	}{
		AssetID:         assetID,
		AssetLabel:      assetLabel,
		Items:           items,
		Services:        serviceStructs,
		Consumables:     consumableStructs,
		ServiceNames:    svcNames,
		ConsumableNames: consNames,
		Message:         message,
		MessageType:     messageType,
	}

	renderTemplate(w, "list.html", data)
}

func createMaintenance(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		http.Error(w, "Missing asset_id in query", http.StatusBadRequest)
		return
	}

	objAssetID, err := primitive.ObjectIDFromHex(assetID)
	if err != nil {
		http.Error(w, "Invalid asset_id", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(w, "create.html", assetID)
		return
	}

	if r.Method == http.MethodPost {
		label := r.FormValue("label")

		doc := MainteneceShedule{
			ID:       primitive.NewObjectID(),
			Lable:    label,
			AssetID:  objAssetID,
			Shedules: []Shedule{},
		}

		ctx, cancel := getCtx()
		defer cancel()

		_, err := db.Collection("maintenances").InsertOne(ctx, doc)
		if err != nil {
			http.Redirect(w, r, "/maintenances?asset_id="+assetID+"&message=Error creating maintenance: "+err.Error()+"&type=error", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/maintenances?asset_id="+assetID+"&message=Maintenance created successfully&type=success", http.StatusSeeOther)
	}
}

// Edit maintenance
func editMaintenance(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}

		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		ctx, cancel := getCtx()
		defer cancel()

		var item MainteneceShedule
		err = db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objID}).Decode(&item)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// Fetch services and consumables from API
		services, consumables := fetchServicesAndConsumables()

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

		data := struct {
			MainteneceShedule
			Services []struct {
				ID    primitive.ObjectID `bson:"_id"`
				Label string             `bson:"label"`
			}
			Consumables []struct {
				ID    primitive.ObjectID `bson:"_id"`
				Label string             `bson:"label"`
			}
		}{
			MainteneceShedule: item,
			Services:          serviceStructs,
			Consumables:       consumableStructs,
		}

		renderTemplate(w, "edit.html", data)
		return
	}

	if r.Method == http.MethodPost {
		idStr := r.FormValue("id")
		if idStr == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}

		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		label := r.FormValue("label")

		ctx, cancel := getCtx()
		defer cancel()

		var item MainteneceShedule
		if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objID}).Decode(&item); err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		_, err = db.Collection("maintenances").UpdateOne(ctx,
			bson.M{"_id": objID},
			bson.M{"$set": bson.M{"label": label}},
		)
		if err != nil {
			http.Error(w, "Update error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/maintenances?asset_id="+item.AssetID.Hex()+"&message=Maintenance updated successfully&type=success", http.StatusSeeOther)
		return
	}
}

// Delete maintenance
func deleteMaintenance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := getCtx()
	defer cancel()

	// Fetch item so we can read AssetID for redirect after delete
	var item MainteneceShedule
	if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objID}).Decode(&item); err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	_, err = db.Collection("maintenances").DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to list with success message
	http.Redirect(w, r, "/maintenances?asset_id="+item.AssetID.Hex()+"&message=Maintenance deleted successfully&type=success", http.StatusSeeOther)
}

// View maintenance with all schedules
func viewMaintenance(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	objID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := getCtx()
	defer cancel()

	var item MainteneceShedule
	if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objID}).Decode(&item); err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Fetch schedules stored separately that reference this maintenance
	scur, err := schedulesCollection.Find(ctx, bson.M{"maintenance_id": objID})
	if err != nil {
		http.Error(w, "Failed to fetch schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer scur.Close(ctx)

	var schedules []ScheduleDoc
	if err := scur.All(ctx, &schedules); err != nil {
		http.Error(w, "Failed to decode schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert ScheduleDoc to Shedule for backward template compatibility
	var shedules []Shedule
	for _, s := range schedules {
		shedules = append(shedules, Shedule{
			ID:          s.ID,
			Lable:       s.Lable,
			SheduleType: s.SheduleType,
			Days:        s.Days,
			Services:    s.Services,
			Consumables: s.Consumables,
			Notes:       s.Notes,
		})
	}

	svcIDs, consIDs := collectScheduleIDs(shedules)

	// Build name maps
	svcNames, consNames := buildNameMaps(ctx, svcIDs, consIDs)

	data := struct {
		MainteneceShedule
		Shedules        []Shedule
		ServiceNames    map[string]string
		ConsumableNames map[string]string
	}{
		MainteneceShedule: item,
		Shedules:          shedules,
		ServiceNames:      svcNames,
		ConsumableNames:   consNames,
	}

	renderTemplate(w, "view.html", data)
}
