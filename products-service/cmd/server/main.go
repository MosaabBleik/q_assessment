package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"database/sql"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/MosaabBleik/products-service/internal/cache"
	"github.com/MosaabBleik/products-service/internal/database"
	"github.com/MosaabBleik/products-service/internal/handlers"
	"github.com/MosaabBleik/products-service/internal/models"
	"github.com/gorilla/mux"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	RunMigrations()

	// Connect to database
	db := database.Connect()

	// Auto migration
	err = db.AutoMigrate(&models.Product{})
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

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func RunMigrations() {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	sslmode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		host, user, password, dbName, port, sslmode,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		log.Fatalf("Migration driver error: %v", err)
	}

	// Resolve migrations folder absolute path
	migrationsPath, err := filepath.Abs("migrations")
	fmt.Println(migrationsPath)
	if err != nil {
		log.Fatalf("Failed to get migrations path: %v", err)
	}
	// Convert backslashes to forward slashes
	migrationsPath = strings.ReplaceAll(migrationsPath, "\\", "/")

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres", driver,
	)
	if err != nil {
		log.Fatalf("Migration init error: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration error: %v", err)
	}

	fmt.Println("✅ Database migrations applied successfully")
}
