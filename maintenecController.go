package main

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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

	// Fetch available services and consumables
	var services []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	}
	var consumables []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	}

	svcCursor, _ := db.Collection("services").Find(ctx, bson.M{})
	_ = svcCursor.All(ctx, &services)
	consCursor, _ := db.Collection("consumables").Find(ctx, bson.M{})
	_ = consCursor.All(ctx, &consumables)

	// Build lookup maps for labels
	svcNames := map[string]string{}
	for _, s := range services {
		svcNames[s.ID.Hex()] = s.Label
	}
	consNames := map[string]string{}
	for _, c := range consumables {
		consNames[c.ID.Hex()] = c.Label
	}

	// Fetch asset label from assets collection
	assetLabel := ""
	var asset struct {
		ID    primitive.ObjectID `bson:"_id"`
		Label string             `bson:"label"`
	}
	err = db.Collection("assets").FindOne(ctx, bson.M{"_id": objAssetID}).Decode(&asset)
	if err == nil {
		assetLabel = asset.Label
	} else {
		// Fallback to asset ID if label not found
		assetLabel = assetID
	}

	// Check for any message to display
	message := r.URL.Query().Get("message")
	messageType := r.URL.Query().Get("type")

	// Pass asset_id, asset_label, items, services, consumables, and lookup maps
	data := struct {
		AssetID         string
		AssetLabel      string
		Items           []MainteneceShedule
		Services        []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}
		Consumables     []struct {
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
		Services:        services,
		Consumables:     consumables,
		ServiceNames:    svcNames,
		ConsumableNames: consNames,
		Message:         message,
		MessageType:     messageType,
	}

	renderTemplate(w, "list.html", data)
}

// Create new maintenance (asset_id comes from URL)
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
		// We no longer need this as we're using a popup now
		// But keeping it for backward compatibility
		renderTemplate(w, "create.html", assetID) // pass asset_id to form
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
			// Handle error with a proper message
			http.Redirect(w, r, "/maintenances?asset_id="+assetID+"&message=Error creating maintenance: "+err.Error()+"&type=error", http.StatusSeeOther)
			return
		}
		
		// Redirect with success message
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

		// Fetch available services and consumables to populate dropdowns
		var services []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}
		var consumables []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}

		svcCursor, _ := db.Collection("services").Find(ctx, bson.M{})
		_ = svcCursor.All(ctx, &services)
		consCursor, _ := db.Collection("consumables").Find(ctx, bson.M{})
		_ = consCursor.All(ctx, &consumables)

		// Build data object that includes the maintenance, services, and consumables
		data := struct {
			MainteneceShedule
			Services    []struct {
				ID    primitive.ObjectID `bson:"_id"`
				Label string             `bson:"label"`
			}
			Consumables []struct {
				ID    primitive.ObjectID `bson:"_id"`
				Label string             `bson:"label"`
			}
		}{
			MainteneceShedule: item,
			Services:          services,
			Consumables:       consumables,
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

		// Fetch item so we can read AssetID for redirect after update
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

		// Redirect back to list with success message
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

// Add schedule
func addShedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parentID := r.FormValue("id")
	objParent, err := primitive.ObjectIDFromHex(parentID)
	if err != nil {
		http.Error(w, "Invalid Parent ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Parse form error: "+err.Error(), http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.FormValue("days"))

	// Parse selected services and consumables (multiple values possible)
	svcVals := r.Form["services[]"]
	consVals := r.Form["consumables[]"]

	var svcIDs []primitive.ObjectID
	for _, s := range svcVals {
		if oid, err := primitive.ObjectIDFromHex(s); err == nil {
			svcIDs = append(svcIDs, oid)
		}
	}

	var consIDs []primitive.ObjectID
	for _, c := range consVals {
		if oid, err := primitive.ObjectIDFromHex(c); err == nil {
			consIDs = append(consIDs, oid)
		}
	}

	shedule := Shedule{
		ID:          primitive.NewObjectID(),
		Lable:       r.FormValue("label"),
		SheduleType: r.FormValue("shedule_type"),
		Days:        days,
		Services:    svcIDs,
		Consumables: consIDs,
		Notes:       r.FormValue("notes"),
	}

	ctx, cancel := getCtx()
	defer cancel()

	// Ensure 'shedules' is an array (not null or missing) before pushing
	ensureFilter := bson.M{"_id": objParent, "$or": []bson.M{{"shedules": bson.M{"$exists": false}}, {"shedules": nil}}}
	ensureUpdate := bson.M{"$set": bson.M{"shedules": []Shedule{}}}
	_, _ = db.Collection("maintenances").UpdateOne(ctx, ensureFilter, ensureUpdate)

	update := bson.M{"$push": bson.M{"shedules": shedule}}
	_, err = db.Collection("maintenances").UpdateByID(ctx, objParent, update)
	if err != nil {
		http.Error(w, "Insert error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the maintenance item to get the asset ID for redirect
	var maintenance MainteneceShedule
	if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objParent}).Decode(&maintenance); err != nil {
		http.Error(w, "Error fetching maintenance: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to list with success message
	http.Redirect(w, r, "/maintenances?asset_id="+maintenance.AssetID.Hex()+"&message=Schedule added successfully&type=success", http.StatusSeeOther)
	return
}

// Delete schedule
func deleteShedule(w http.ResponseWriter, r *http.Request) {
	parentID := r.URL.Query().Get("id")
	sid := r.URL.Query().Get("sid")

	objParent, err := primitive.ObjectIDFromHex(parentID)
	if err != nil {
		http.Error(w, "Invalid Parent ID", http.StatusBadRequest)
		return
	}
	objShedule, err := primitive.ObjectIDFromHex(sid)
	if err != nil {
		http.Error(w, "Invalid Shedule ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := getCtx()
	defer cancel()

	update := bson.M{"$pull": bson.M{"shedules": bson.M{"_id": objShedule}}}
	_, err = db.Collection("maintenances").UpdateOne(ctx, bson.M{"_id": objParent}, update)
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/maintenances/edit?id="+parentID, http.StatusSeeOther)
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

	// Collect all referenced service and consumable IDs from schedules
	svcIDSet := map[primitive.ObjectID]struct{}{}
	consIDSet := map[primitive.ObjectID]struct{}{}
	for _, s := range item.Shedules {
		for _, sid := range s.Services {
			svcIDSet[sid] = struct{}{}
		}
		for _, cid := range s.Consumables {
			consIDSet[cid] = struct{}{}
		}
	}

	// Helper to convert set to slice for query
	var svcIDs []primitive.ObjectID
	for id := range svcIDSet {
		svcIDs = append(svcIDs, id)
	}
	var consIDs []primitive.ObjectID
	for id := range consIDSet {
		consIDs = append(consIDs, id)
	}

	// Fetch names for services
	svcNames := map[string]string{}
	if len(svcIDs) > 0 {
		cursor, err := db.Collection("services").Find(ctx, bson.M{"_id": bson.M{"$in": svcIDs}})
		if err == nil {
			var rows []struct {
				ID   primitive.ObjectID `bson:"_id"`
				Name string             `bson:"name"`
			}
			if err := cursor.All(ctx, &rows); err == nil {
				for _, r := range rows {
					svcNames[r.ID.Hex()] = r.Name
				}
			}
		}
	}

	// Fetch names for consumables
	consNames := map[string]string{}
	if len(consIDs) > 0 {
		cursor, err := db.Collection("consumables").Find(ctx, bson.M{"_id": bson.M{"$in": consIDs}})
		if err == nil {
			var rows []struct {
				ID   primitive.ObjectID `bson:"_id"`
				Name string             `bson:"name"`
			}
			if err := cursor.All(ctx, &rows); err == nil {
				for _, r := range rows {
					consNames[r.ID.Hex()] = r.Name
				}
			}
		}
	}

	// Render view with lookup maps available to template (keys are hex strings)
	data := struct {
		MainteneceShedule
		ServiceNames    map[string]string
		ConsumableNames map[string]string
	}{
		MainteneceShedule: item,
		ServiceNames:      svcNames,
		ConsumableNames:   consNames,
	}

	renderTemplate(w, "view.html", data)
}