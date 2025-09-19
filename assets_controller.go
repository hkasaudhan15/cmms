package main

import (
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

func getAssets(db *mongo.Database) http.HandlerFunc {
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

        templates.ExecuteTemplate(w, "Login.html", result)
    }
}

func addAsset(db *mongo.Database) http.HandlerFunc {
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
