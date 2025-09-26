package internal

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var templates *template.Template

func init() {
	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).ParseGlob("templates/*.html"))
}

// GetAssets renders all asset records on the asset page
func GetAssets(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var result AssetsPageData

		data, err := getAllAssets(ctx, db)
		if err != nil {
			log.Printf("error fetching records: %v", err)
			result.Error = "Error fetching records"
		} else {
			result.Data = data
		}

		if msg := r.URL.Query().Get("success"); msg != "" {
			result.Message = msg
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			result.Error = errMsg
		}

		if err := templates.ExecuteTemplate(w, "Asset.html", result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// AddAsset inserts a new asset record into the database
func AddAsset(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		asset := Asset{
			Label:    r.FormValue("label"),
			Type:     r.FormValue("type"),
			Location: r.FormValue("location"),
		}

		dateStr := r.FormValue("effective_date")
		if dateStr == "" {
			http.Redirect(w, r, "/assets?error=Effective+date+required", http.StatusSeeOther)
			return
		}

		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Redirect(w, r, "/assets?error=Invalid+date+format", http.StatusSeeOther)
			return
		}
		asset.EffectiveDate = parsedDate

		err = insertAsset(ctx, db, asset)
		if err != nil {
			http.Redirect(w, r, "/assets?error=Failed+to+insert+asset", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/assets?success=Asset+added+successfully!", http.StatusSeeOther)
	}
}

// EditAsset updates an existing asset record identified by its ID
func EditAsset(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		idStr := vars["id"]

		objID, _ := primitive.ObjectIDFromHex(idStr)

		label := r.FormValue("label")
		typ := r.FormValue("type")
		location := r.FormValue("location")
		dateStr := r.FormValue("effective_date")

		effectiveDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Redirect(w, r, "/assets?error=Invalid+date", http.StatusSeeOther)
			return
		}

		asset := Asset{
			Label:         label,
			Type:          typ,
			Location:      location,
			EffectiveDate: effectiveDate,
		}

		err = updateAsset(ctx, db, objID, asset)
		if err != nil {
			http.Redirect(w, r, "/assets?error=Error+updating+asset", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/assets?success=Asset+updated+successfully", http.StatusSeeOther)
	}
}

// DeleteAsset deletes an existing asset record by its ID
func DeleteAsset(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		idStr := vars["id"]

		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Redirect(w, r, "/assets?error=Invalid+asset+ID", http.StatusSeeOther)
			return
		}

		err = deleteAssetByID(ctx, db, objID)
		if err != nil {
			http.Redirect(w, r, "/assets?error=Failed+to+delete+asset", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/assets?success=Asset+deleted+successfully", http.StatusSeeOther)
	}
}

// GetAsset returns a single asset by its ID in JSON format
func GetAsset(db *mongo.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := mux.Vars(r)
		idStr := vars["id"]

		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			http.Error(w, "Invalid ID format", http.StatusBadRequest)
			return
		}

		asset, err := getAssetByID(ctx, db, objID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				http.Error(w, "Asset not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(asset); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
