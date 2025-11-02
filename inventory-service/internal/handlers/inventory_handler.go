package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	product_clients "github.com/MosaabBleik/inventory-service/internal/clients"
	"github.com/MosaabBleik/inventory-service/internal/database"
	"github.com/MosaabBleik/inventory-service/internal/models"
)

type InventoryHandler struct {
	ProductsClient *product_clients.ProductsClient
}

func (h *InventoryHandler) AddInventoryHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			ProductID         string `json:"product_id"`
			WarehouseLocation string `json:"warehouse_location"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		product, err := h.ProductsClient.GetProduct(ctx, req.ProductID)
		if err != nil {
			http.Error(w, "unable to verify product: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		fmt.Println(product)

		inventory := models.Inventory{
			ProductID:         req.ProductID,
			WarehouseLocation: req.WarehouseLocation,
			Quantity:          0,
		}

		if err := database.DB.Create(&inventory).Error; err != nil {
			http.Error(w, fmt.Sprintf("Error creating inventory: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(product); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (h *InventoryHandler) StockHandler(w http.ResponseWriter, r *http.Request) {
	const prefixToTrim = "/api/products"

	idStr := strings.TrimPrefix(r.URL.Path, prefixToTrim)
	idStr = strings.TrimPrefix(idStr, "/")
	productID := strings.TrimSpace(idStr)

	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	product, err := h.ProductsClient.GetProduct(ctx, productID)
	if err != nil {
		http.Error(w, "unable to verify product: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	fmt.Println(product)

	switch r.Method {
	case http.MethodGet:
		var inventory models.Inventory
		err := database.DB.
			Where("product_id = ?", productID).
			First(&inventory).Error

		if err != nil {
			http.Error(w, "Inventory not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(inventory); err != nil {
			http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		}
		return
	case http.MethodPut:
		var req struct {
			ProductID         string `json:"product_id"`
			WarehouseLocation string `json:"warehouse_location"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}

func (h *InventoryHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var req CheckAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Call the Products Service via client
	ctx := context.Background()
	product, err := h.ProductsClient.GetProduct(ctx, req.ProductID)
	if err != nil {
		http.Error(w, "unable to verify product: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Example: respond with product info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"product": product,
	})
}
