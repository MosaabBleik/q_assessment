package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/MosaabBleik/products-service/internal/database"
	"github.com/MosaabBleik/products-service/internal/handlers"
	"github.com/MosaabBleik/products-service/internal/models"
)

func main() {
	// Connect to database
	db := database.Connect()

	// Auto migration
	err := db.AutoMigrate(&models.Product{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Define endpoints
	http.HandleFunc("/api/products", handlers.ProductHandler)
	http.HandleFunc("/api/products/", handlers.ProductHandler)
	http.HandleFunc("/api/products/search", handlers.SearchProductHandler)
	http.HandleFunc("/api/products/bulk-update", handlers.BulkUpdateHandler)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
