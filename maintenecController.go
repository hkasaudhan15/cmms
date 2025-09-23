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

// Helper function to fetch services and consumables
func fetchServicesAndConsumables(ctx context.Context) ([]struct {
	ID    primitive.ObjectID `bson:"_id"`
	Label string             `bson:"label"`
}, []struct {
	ID    primitive.ObjectID `bson:"_id"`
	Label string             `bson:"label"`
}) {
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

	return services, consumables
}

// Helper function to build service and consumable name maps
func buildNameMaps(ctx context.Context, svcIDs, consIDs []primitive.ObjectID) (map[string]string, map[string]string) {
	// Fetch names for services
	svcNames := map[string]string{}
	if len(svcIDs) > 0 {
		cursor, err := db.Collection("services").Find(ctx, bson.M{"_id": bson.M{"$in": svcIDs}})
		if err == nil {
			var rows []struct {
				ID    primitive.ObjectID `bson:"_id"`
				Label string             `bson:"label"`
			}
			if err := cursor.All(ctx, &rows); err == nil {
				for _, r := range rows {
					svcNames[r.ID.Hex()] = r.Label
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
				ID    primitive.ObjectID `bson:"_id"`
				Label string             `bson:"label"`
			}
			if err := cursor.All(ctx, &rows); err == nil {
				for _, r := range rows {
					consNames[r.ID.Hex()] = r.Label
				}
			}
		}
	}

	return svcNames, consNames
}

// Helper function to collect service and consumable IDs from schedules
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

	// Helper to convert set to slice for query
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

// Helper function to get asset label
func getAssetLabel(ctx context.Context, assetID primitive.ObjectID) string {
	var asset Asset
	err := db.Collection("assets").FindOne(ctx, bson.M{"_id": assetID}).Decode(&asset)
	if err == nil && asset.Label != "" {
		return asset.Label
	} else if err == nil {
		// Asset found but label is empty, fallback to ID
		return assetID.Hex()
	} else {
		// Asset not found, fallback to ID
		return assetID.Hex()
	}
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
	services, consumables := fetchServicesAndConsumables(ctx)

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
		services, consumables := fetchServicesAndConsumables(ctx)

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

// List schedules for an asset
func listSchedules(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Failed to fetch maintenances: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var maintenances []MainteneceShedule
	if err = cursor.All(ctx, &maintenances); err != nil {
		http.Error(w, "Failed to decode maintenances: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(maintenances) == 0 {
		http.Error(w, "No maintenances found for asset", http.StatusNotFound)
		return
	}

	var allSchedules []Shedule
	var svcIDs, consIDs []primitive.ObjectID

	for _, m := range maintenances {
		allSchedules = append(allSchedules, m.Shedules...)

		sIDs, cIDs := collectScheduleIDs(m.Shedules)
		svcIDs = append(svcIDs, sIDs...)
		consIDs = append(consIDs, cIDs...)
	}

	// Build name maps for services & consumables
	svcNames, consNames := buildNameMaps(ctx, svcIDs, consIDs)

	// Fetch asset label for display
	assetLabel := getAssetLabel(ctx, objAssetID)

	// Check for any message to display
	message := r.URL.Query().Get("message")
	messageType := r.URL.Query().Get("type")

	services, consumables := fetchServicesAndConsumables(ctx)

	data := struct {
		Maintenances []MainteneceShedule
		Schedules    []Shedule
		AssetLabel   string
		Services     []struct {
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
		Maintenances:    maintenances,
		Schedules:       allSchedules,
		AssetLabel:      assetLabel,
		Services:        services,
		Consumables:     consumables,
		ServiceNames:    svcNames,
		ConsumableNames: consNames,
		Message:         message,
		MessageType:     messageType,
	}

	renderTemplate(w, "schedule_list.html", data)
}

// Add schedule
func addSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maintenanceID := r.FormValue("maintenance_id")
	objMaintenance, err := primitive.ObjectIDFromHex(maintenanceID)
	if err != nil {
		http.Error(w, "Invalid Maintenance ID", http.StatusBadRequest)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Parse form error: "+err.Error(), http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.FormValue("days"))

	// Parse selected services and consumables
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

	ensureFilter := bson.M{"_id": objMaintenance, "$or": []bson.M{{"shedules": bson.M{"$exists": false}}, {"shedules": nil}}}
	ensureUpdate := bson.M{"$set": bson.M{"shedules": []Shedule{}}}
	_, _ = db.Collection("maintenances").UpdateOne(ctx, ensureFilter, ensureUpdate)

	update := bson.M{"$push": bson.M{"shedules": shedule}}
	_, err = db.Collection("maintenances").UpdateByID(ctx, objMaintenance, update)
	if err != nil {
		http.Error(w, "Insert error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var maintenance MainteneceShedule
	if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objMaintenance}).Decode(&maintenance); err != nil {
		http.Error(w, "Maintenance not found for redirect", http.StatusInternalServerError)
		return
	}

	// Redirect with asset_id
	http.Redirect(w, r, "/schedules?asset_id="+maintenance.AssetID.Hex()+"&message=Schedule added successfully&type=success", http.StatusSeeOther)
}

// Edit schedule
func editSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maintenanceID := r.FormValue("maintenance_id")
	scheduleID := r.FormValue("schedule_id")

	objMaintenance, err := primitive.ObjectIDFromHex(maintenanceID)
	if err != nil {
		http.Error(w, "Invalid Maintenance ID", http.StatusBadRequest)
		return
	}

	objSchedule, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		http.Error(w, "Invalid Schedule ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Parse form error: "+err.Error(), http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.FormValue("days"))

	// Services & consumables
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

	ctx, cancel := getCtx()
	defer cancel()

	filter := bson.M{"_id": objMaintenance, "shedules._id": objSchedule}
	update := bson.M{
		"$set": bson.M{
			"shedules.$.label":        r.FormValue("label"),
			"shedules.$.shedule_type": r.FormValue("shedule_type"),
			"shedules.$.days":         days,
			"shedules.$.services":     svcIDs,
			"shedules.$.consumables":  consIDs,
			"shedules.$.notes":        r.FormValue("notes"),
		},
	}

	_, err = db.Collection("maintenances").UpdateOne(ctx, filter, update)
	if err != nil {
		http.Error(w, "Update error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var maintenance MainteneceShedule
	if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objMaintenance}).Decode(&maintenance); err != nil {
		http.Error(w, "Maintenance not found for redirect", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/schedules?asset_id="+maintenance.AssetID.Hex()+"&message=Schedule updated successfully&type=success", http.StatusSeeOther)
}

// Delete schedule
func deleteSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maintenanceID := r.FormValue("maintenance_id")
	scheduleID := r.FormValue("schedule_id")

	objMaintenance, err := primitive.ObjectIDFromHex(maintenanceID)
	if err != nil {
		http.Error(w, "Invalid Maintenance ID", http.StatusBadRequest)
		return
	}

	objSchedule, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		http.Error(w, "Invalid Schedule ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := getCtx()
	defer cancel()

	filter := bson.M{"_id": objMaintenance}
	update := bson.M{"$pull": bson.M{"shedules": bson.M{"_id": objSchedule}}}

	_, err = db.Collection("maintenances").UpdateOne(ctx, filter, update)
	if err != nil {
		http.Error(w, "Delete error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var maintenance MainteneceShedule
	if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": objMaintenance}).Decode(&maintenance); err != nil {
		http.Error(w, "Maintenance not found for redirect", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/schedules?asset_id="+maintenance.AssetID.Hex()+"&message=Schedule deleted successfully&type=success", http.StatusSeeOther)
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
	svcIDs, consIDs := collectScheduleIDs(item.Shedules)

	// Build name maps
	svcNames, consNames := buildNameMaps(ctx, svcIDs, consIDs)

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

func addShedule(w http.ResponseWriter, r *http.Request) {
	addSchedule(w, r)
}

func deleteShedule(w http.ResponseWriter, r *http.Request) {
	deleteSchedule(w, r)
}
