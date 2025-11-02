package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	product_clients "github.com/MosaabBleik/inventory-service/internal/clients"
	"github.com/MosaabBleik/inventory-service/internal/database"
	"github.com/MosaabBleik/inventory-service/internal/handlers"
	"github.com/MosaabBleik/inventory-service/internal/models"
)

func main() {
	// Connect to database
	db := database.Connect()

	// Auto migration
	err := db.AutoMigrate(&models.Inventory{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	productsURL := os.Getenv("PRODUCTS_SERVICE_URL")
	if productsURL == "" {
		productsURL = "http://products-service:8080"
	}

	// Create the client
	productsClient := product_clients.NewProductsClient(productsURL)

	// Create handler and inject client
	inventoryHandler := &handlers.InventoryHandler{
		ProductsClient: productsClient,
	}

	// Define endpoints
	http.HandleFunc("/api/inventory", inventoryHandler.AddInventoryHandler)
	http.HandleFunc("/api/inventory/", inventoryHandler.StockHandler)
	http.HandleFunc("/api/products/low-stock", inventoryHandler.CheckAvailability)
	http.HandleFunc("/api/products/check-availability", inventoryHandler.CheckAvailability)
	http.HandleFunc("/api/health", inventoryHandler.CheckAvailability)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
