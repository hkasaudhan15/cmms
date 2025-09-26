package main

import (
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

	// Lists schedules stored as top-level documents in the schedules collection
	cursor, err := schedulesCollection.Find(ctx, bson.M{"asset_id": objAssetID})
	if err != nil {
		http.Error(w, "Failed to fetch schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var scheduleDocs []ScheduleDoc
	if err = cursor.All(ctx, &scheduleDocs); err != nil {
		http.Error(w, "Failed to decode schedules: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Build helper maps and lists
	var svcIDs, consIDs []primitive.ObjectID
	for _, s := range scheduleDocs {
		svcIDs = append(svcIDs, s.Services...)
		consIDs = append(consIDs, s.Consumables...)
	}

	svcNames, consNames := buildNameMaps(ctx, svcIDs, consIDs)

	assetLabel := getAssetLabel(ctx, objAssetID)

	message := r.URL.Query().Get("message")
	messageType := r.URL.Query().Get("type")

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

	// We also need maintenances list for the dropdown; fetch from maintenances collection
	mcursor, err := db.Collection("maintenances").Find(ctx, bson.M{"asset_id": objAssetID})
	if err != nil {
		http.Error(w, "Failed to fetch maintenances: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer mcursor.Close(ctx)

	var maintenances []MainteneceShedule
	if err := mcursor.All(ctx, &maintenances); err != nil {
		http.Error(w, "Failed to decode maintenances: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// build maintenance id -> label map for quick lookup in template
	maintMap := map[string]string{}
	for _, m := range maintenances {
		maintMap[m.ID.Hex()] = m.Lable
	}

	data := struct {
		Maintenances []MainteneceShedule
		Schedules    []ScheduleDoc
		AssetID      string
		AssetLabel   string
		Services     []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}
		Consumables []struct {
			ID    primitive.ObjectID `bson:"_id"`
			Label string             `bson:"label"`
		}
		MaintMap        map[string]string
		ServiceNames    map[string]string
		ConsumableNames map[string]string
		Message         string
		MessageType     string
	}{
		Maintenances:    maintenances,
		Schedules:       scheduleDocs,
		AssetID:         assetID,
		AssetLabel:      assetLabel,
		Services:        serviceStructs,
		Consumables:     consumableStructs,
		ServiceNames:    svcNames,
		ConsumableNames: consNames,
		MaintMap:        maintMap,
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

	// Maintenance ID is optional: schedules are stored per-asset and may be linked to a maintenance
	maintenanceID := r.FormValue("maintenance_id")
	var objMaintenance *primitive.ObjectID
	if maintenanceID != "" {
		if oid, err := primitive.ObjectIDFromHex(maintenanceID); err == nil {
			objMaintenance = &oid
		} else {
			http.Error(w, "Invalid Maintenance ID", http.StatusBadRequest)
			return
		}
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Parse form error: "+err.Error(), http.StatusBadRequest)
		return
	}

	days, _ := strconv.Atoi(r.FormValue("days"))

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

	shedule := ScheduleDoc{
		ID:            primitive.NewObjectID(),
		MaintenanceID: objMaintenance,
		AssetID:       primitive.NilObjectID, // set below
		Lable:         r.FormValue("label"),
		SheduleType:   r.FormValue("shedule_type"),
		Days:          days,
		Services:      svcIDs,
		Consumables:   consIDs,
		Notes:         r.FormValue("notes"),
	}

	// AssetID must be determined from maintenance (if given) or form param asset_id
	assetID := r.FormValue("asset_id")
	if assetID == "" {
		if objMaintenance != nil {
			// lookup maintenance to get asset id
			ctx, cancel := getCtx()
			defer cancel()
			var m MainteneceShedule
			if err := db.Collection("maintenances").FindOne(ctx, bson.M{"_id": *objMaintenance}).Decode(&m); err != nil {
				http.Error(w, "Maintenance not found to resolve asset_id", http.StatusInternalServerError)
				return
			}
			shedule.AssetID = m.AssetID
		} else {
			http.Error(w, "Missing asset_id", http.StatusBadRequest)
			return
		}
	} else {
		if aoid, err := primitive.ObjectIDFromHex(assetID); err == nil {
			shedule.AssetID = aoid
		} else {
			http.Error(w, "Invalid asset_id", http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := getCtx()
	defer cancel()

	if _, err := schedulesCollection.InsertOne(ctx, shedule); err != nil {
		http.Error(w, "Insert error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect with asset_id
	http.Redirect(w, r, "/schedules?asset_id="+shedule.AssetID.Hex()+"&message=Schedule added successfully&type=success", http.StatusSeeOther)
}

// Edit schedule
func editSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scheduleID := r.FormValue("schedule_id")
	if scheduleID == "" {
		http.Error(w, "Missing schedule_id", http.StatusBadRequest)
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

	// Update schedule document in schedules collection
	filter := bson.M{"_id": objSchedule}
	update := bson.M{"$set": bson.M{
		"label":        r.FormValue("label"),
		"shedule_type": r.FormValue("shedule_type"),
		"days":         days,
		"services":     svcIDs,
		"consumables":  consIDs,
		"notes":        r.FormValue("notes"),
	}}

	if _, err := schedulesCollection.UpdateOne(ctx, filter, update); err != nil {
		http.Error(w, "Update error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// find schedule to get asset id for redirect
	var updated ScheduleDoc
	if err := schedulesCollection.FindOne(ctx, bson.M{"_id": objSchedule}).Decode(&updated); err != nil {
		http.Error(w, "Schedule not found for redirect", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/schedules?asset_id="+updated.AssetID.Hex()+"&message=Schedule updated successfully&type=success", http.StatusSeeOther)
}

// Delete schedule
func deleteSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scheduleID := r.FormValue("schedule_id")
	if scheduleID == "" {
		http.Error(w, "Missing schedule_id", http.StatusBadRequest)
		return
	}
	objSchedule, err := primitive.ObjectIDFromHex(scheduleID)
	if err != nil {
		http.Error(w, "Invalid Schedule ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := getCtx()
	defer cancel()

	// find schedule first to get asset id for redirect
	var sched ScheduleDoc
	if err := schedulesCollection.FindOne(ctx, bson.M{"_id": objSchedule}).Decode(&sched); err != nil {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	}

	if _, err := schedulesCollection.DeleteOne(ctx, bson.M{"_id": objSchedule}); err != nil {
		http.Error(w, "Delete error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/schedules?asset_id="+sched.AssetID.Hex()+"&message=Schedule deleted successfully&type=success", http.StatusSeeOther)
}

func addShedule(w http.ResponseWriter, r *http.Request) {
	addSchedule(w, r)
}

func deleteShedule(w http.ResponseWriter, r *http.Request) {
	deleteSchedule(w, r)
}
