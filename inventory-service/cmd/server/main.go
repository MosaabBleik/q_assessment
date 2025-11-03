package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	product_clients "github.com/MosaabBleik/inventory-service/internal/clients"
	"github.com/MosaabBleik/inventory-service/internal/database"
	"github.com/MosaabBleik/inventory-service/internal/handlers"
	"github.com/MosaabBleik/inventory-service/internal/middleware"
	"github.com/MosaabBleik/inventory-service/internal/models"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	db := database.Connect()

	// Auto migration
	err := db.AutoMigrate(&models.Inventory{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	productsURL := os.Getenv("PRODUCTS_SERVICE_URL")
	if productsURL == "" {
		productsURL = "http://localhost:8080"
	}

	timeoutStr := os.Getenv("REQUEST_TIMEOUT")
	timeoutSeconds, err := strconv.Atoi(timeoutStr)
	if err != nil {
		// Use default value
		timeoutSeconds = 5
	}

	// Create the client
	productsClient := product_clients.NewProductsClient(productsURL, timeoutSeconds)

	inventoryHandler := &handlers.InventoryHandler{
		DB:             db,
		ProductsClient: productsClient,
	}

	// Router
	r := mux.NewRouter()

	// Define endpoints
	r.HandleFunc("/api/inventory", inventoryHandler.AddInventory).Methods("POST")
	r.HandleFunc("/api/inventory/low-stock", inventoryHandler.LowStock).Methods("GET")
	r.HandleFunc("/api/inventory/{product_id}", inventoryHandler.Stock).Methods("GET")
	r.HandleFunc("/api/inventory/{product_id}", inventoryHandler.UpdateStock).Methods("PUT")
	r.HandleFunc("/api/inventory/check-availability", inventoryHandler.CheckAvailability).Methods("POST")
	r.HandleFunc("/api/health", inventoryHandler.HealthCheck).Methods("GET")

	loggedRouter := middleware.Logger(r)

	port := os.Getenv("PORT")
	portStr := fmt.Sprintf(":%s", port)

	fmt.Println("Server started at :", port)
	log.Fatal(http.ListenAndServe(portStr, loggedRouter))
}
