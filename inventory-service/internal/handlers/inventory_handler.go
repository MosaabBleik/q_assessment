package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	product_clients "github.com/MosaabBleik/inventory-service/internal/clients"
	"github.com/MosaabBleik/inventory-service/internal/models"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type CheckAvailabilityRequest struct {
	Items []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

type WarehouseStock struct {
	Location string `json:"location"`
	Quantity int    `json:"quantity"`
}

type ItemAvailability struct {
	ProductID      string           `json:"product_id"`
	Requested      int              `json:"requested"`
	AvailableStock int              `json:"available_stock"`
	Status         string           `json:"status"`
	StatusCode     int              `json:"status_code,omitempty"`
	Warehouses     []WarehouseStock `json:"warehouses,omitempty"`
}

type CheckAvailabilityResponse struct {
	Available bool               `json:"available"`
	Items     []ItemAvailability `json:"items"`
}

type InventoryHandler struct {
	DB             *gorm.DB
	ProductsClient *product_clients.ProductsClient
}

func (h *InventoryHandler) AddInventory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductID         string `json:"product_id"`
		WarehouseLocation string `json:"warehouse_location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, prodStatus, err := h.ProductsClient.GetProduct(ctx, req.ProductID)
	if err != nil {
		WriteProductErrorResponse(w, prodStatus, err)
		return
	}

	var existing models.Inventory
	err = h.DB.WithContext(ctx).
		Where("product_id = ? AND warehouse_location = ?", req.ProductID, req.WarehouseLocation).
		First(&existing).Error

	if err == nil {
		http.Error(w, "inventory already exists for this product and warehouse", http.StatusConflict)
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, fmt.Sprintf("error checking inventory: %s", err), http.StatusInternalServerError)
		return
	}

	inventory := models.Inventory{
		ProductID:         req.ProductID,
		WarehouseLocation: req.WarehouseLocation,
		Quantity:          0,
	}

	if err := h.DB.WithContext(ctx).Create(&inventory).Error; err != nil {
		http.Error(w, fmt.Sprintf("Error creating inventory: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(inventory); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *InventoryHandler) Stock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["product_id"]

	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, prodStatus, err := h.ProductsClient.GetProduct(ctx, productID)
	if err != nil {
		WriteProductErrorResponse(w, prodStatus, err)
		return
	}

	// Fetch all inventories for the given product
	var inventories []models.Inventory
	err = h.DB.WithContext(ctx).
		Where("product_id = ?", productID).
		Find(&inventories).Error

	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	if len(inventories) == 0 {
		http.Error(w, "no inventory records found for this product", http.StatusNotFound)
		return
	}

	// Calculate total quantity
	totalQuantity := 0
	for _, inv := range inventories {
		totalQuantity += inv.Quantity
	}

	// Response payload
	response := map[string]any{
		"product_id":     productID,
		"total_quantity": totalQuantity,
		"inventories":    inventories,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}

func (h *InventoryHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	productID := vars["product_id"]

	if productID == "" {
		http.Error(w, "Product ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Quantity          int    `json:"quantity"`
		WarehouseLocation string `json:"warehouse_location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.WarehouseLocation == "" {
		http.Error(w, "warehouse_location is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, prodStatus, err := h.ProductsClient.GetProduct(ctx, productID)
	if err != nil {
		WriteProductErrorResponse(w, prodStatus, err)
		return
	}

	var inventory models.Inventory
	err = h.DB.WithContext(ctx).
		Where("product_id = ? AND warehouse_location = ?", productID, req.WarehouseLocation).
		First(&inventory).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		http.Error(w, "inventory record not found for given product and warehouse", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	inventory.Quantity += req.Quantity
	if err := h.DB.WithContext(ctx).Save(&inventory).Error; err != nil {
		http.Error(w, fmt.Sprintf("failed to update stock: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "stock updated successfully",
		"product":   inventory.ProductID,
		"warehouse": inventory.WarehouseLocation,
		"quantity":  inventory.Quantity,
	})
}

func (h *InventoryHandler) LowStock(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var lowStockItems []models.Inventory

	if err := h.DB.WithContext(ctx).
		Where("quantity < ?", 10).
		Find(&lowStockItems).Error; err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"count": len(lowStockItems),
		"items": lowStockItems,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}

func (h *InventoryHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var req CheckAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	results := make([]ItemAvailability, len(req.Items))

	var wg sync.WaitGroup
	wg.Add(len(req.Items))

	for i, item := range req.Items {
		go func(i int, item struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		}) {
			defer wg.Done()

			productCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			// Check if product exists via Products Service
			_, prodStatus, err := h.ProductsClient.GetProduct(productCtx, item.ProductID)
			if err != nil {
				errMsg := ""
				switch prodStatus {
				case 4:
					errMsg = "product_not_found"
				case 2, 5:
					errMsg = "products_service_not_available"
				}
				results[i] = ItemAvailability{
					ProductID:      item.ProductID,
					Requested:      item.Quantity,
					AvailableStock: 0,
					StatusCode:     prodStatus,
					Status:         errMsg,
				}
				return
			}

			// Check inventory stock
			var inventories []struct {
				WarehouseLocation string
				Quantity          int
			}

			err = h.DB.WithContext(productCtx).
				Table("inventories").
				Select("warehouse_location, quantity").
				Where("product_id = ?", item.ProductID).
				Scan(&inventories).Error

			if err != nil {
				results[i] = ItemAvailability{
					ProductID:      item.ProductID,
					Requested:      item.Quantity,
					AvailableStock: 0,
					Status:         fmt.Sprintf("error_checking_stock: %e", err),
				}
				return
			}

			// Step 3: Sum total stock and prepare warehouse breakdown
			totalStock := 0
			var warehouses []WarehouseStock
			for _, inv := range inventories {
				totalStock += inv.Quantity
				warehouses = append(warehouses, WarehouseStock{
					Location: inv.WarehouseLocation,
					Quantity: inv.Quantity,
				})
			}

			status := "available"
			if totalStock < item.Quantity {
				status = "insufficient_stock"
			}

			results[i] = ItemAvailability{
				ProductID:      item.ProductID,
				Requested:      item.Quantity,
				AvailableStock: totalStock,
				Status:         status,
				Warehouses:     warehouses,
			}
		}(i, item)
	}

	wg.Wait()

	// Determine overall availability
	allAvailable := true
	for _, res := range results {
		if res.Status != "available" {
			allAvailable = false
			break
		}
	}

	resp := CheckAvailabilityResponse{
		Available: allAvailable,
		Items:     results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *InventoryHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	dbStatus := "ok"

	if err := h.DB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		dbStatus = "error"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"database": dbStatus,
	})
}

func WriteProductErrorResponse(w http.ResponseWriter, prodStatus int, err error) {
	w.Header().Set("Content-Type", "application/json")

	var statusCode int
	var msg string

	switch prodStatus {
	case 1, 6: // request creation failed / unknown error
		statusCode = http.StatusInternalServerError
		msg = err.Error()
	case 2: // timeouts
		statusCode = http.StatusGatewayTimeout
		msg = err.Error()
	case 5: // network errors / timeouts
		statusCode = http.StatusServiceUnavailable
		msg = err.Error()
	case 3: // context canceled
		statusCode = http.StatusRequestTimeout
		msg = err.Error()
	case 4: // product not found
		statusCode = http.StatusNotFound
		msg = err.Error()
	default:
		statusCode = http.StatusInternalServerError
		msg = err.Error()
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}
