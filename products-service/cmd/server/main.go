package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/MosaabBleik/products-service/internal/cache"
	"github.com/MosaabBleik/products-service/internal/database"
	"github.com/MosaabBleik/products-service/internal/handlers"
	"github.com/MosaabBleik/products-service/internal/middleware"
	"github.com/MosaabBleik/products-service/internal/models"
	"github.com/gorilla/mux"
)

func main() {
	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	db := database.Connect()

	// Auto migration
	err := db.AutoMigrate(&models.Product{})
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Cache Redis Client
	redisClient, err := cache.InitRedis()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	productHandler := handlers.ProductHandler{
		DB:          db,
		RedisClient: redisClient,
	}

	// Router
	r := mux.NewRouter()

	// Advanced search
	r.HandleFunc("/api/products/search", productHandler.Search).Methods("GET")

	// CRUD handlers
	r.HandleFunc("/api/products", productHandler.ListProducts).Methods("GET")
	r.HandleFunc("/api/products", productHandler.CreateProduct).Methods("POST")
	r.HandleFunc("/api/products/{id}", productHandler.GetProduct).Methods("GET")
	r.HandleFunc("/api/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	r.HandleFunc("/api/products/{id}", productHandler.DeleteProduct).Methods("DELETE")

	// Bulk update
	r.HandleFunc("/api/products/bulk-update", productHandler.BulkUpdate).Methods("POST")

	// Health check
	r.HandleFunc("/api/health", productHandler.HealthCheck).Methods("GET")

	loggedRouter := middleware.Logger(r)

	port := os.Getenv("PORT")
	portStr := fmt.Sprintf(":%s", port)

	fmt.Println("Server started at :", port)
	log.Fatal(http.ListenAndServe(portStr, loggedRouter))
}
