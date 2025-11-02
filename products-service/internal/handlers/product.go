package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MosaabBleik/products-service/internal/database"
	"github.com/MosaabBleik/products-service/internal/models"
)

const (
	defaultPage  = 1
	defaultLimit = 10
)

func getPaginationParams(r *http.Request) (page int, limit int) {
	/*Get the products pagination (Default 1-10)*/

	// 1. Get query parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// --- Set Defaults Pagination Values ---
	page = defaultPage
	limit = defaultLimit

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			// Check if the parsed value is positive before setting
			if p > 0 {
				page = p
			}
		}
	}

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 0 && l <= 100 {
				limit = l
			} else if l > 100 {
				limit = 100
			}
		}
	}

	return page, limit
}

func ProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get the last part of the URL ("/products/241" => "241")
	const prefixToTrim = "/api/products"

	idStr := strings.TrimPrefix(r.URL.Path, prefixToTrim)
	idStr = strings.TrimPrefix(idStr, "/")
	productID := strings.TrimSpace(idStr)

	isListRequest := productID == ""

	switch r.Method {
	case http.MethodGet:

		//##// Return Single Product
		if !isListRequest {
			fmt.Println(productID)
			var product models.Product
			err := database.DB.
				Where("id = ?", productID).
				First(&product).Error

			if err != nil {
				http.Error(w, "Product not found", http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(product); err != nil {
				http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
			}
			return
		}

		page, limit := getPaginationParams(r)

		// Enforce positive pagination
		if page <= 0 {
			page = 1
		}
		if limit <= 0 {
			limit = 10
		}

		query := database.DB.Model(&models.Product{})

		// Pagination
		offset := (page - 1) * limit
		query = query.Limit(limit).Offset(offset)

		// Fetch Products
		var products []models.Product
		if err := query.Find(&products).Error; err != nil {
			http.Error(w, "failed to fetch products", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"count":    len(products),
			"products": products,
		})

	case http.MethodPost:
		type CreateProductRequest struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Price       float64 `json:"price"`
			Category    string  `json:"category"`
		}
		req := CreateProductRequest{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var product models.Product

		product.Name = req.Name
		product.Description = req.Description
		product.Price = req.Price
		product.Category = req.Category

		if err := database.DB.Create(&product).Error; err != nil {
			http.Error(w, fmt.Sprintf("Error creating products: %s", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(product); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}

	case http.MethodPut:
		if idStr == "" || idStr == "/" {
			http.Error(w, "Product ID is missing", http.StatusBadRequest)
			return
		}

		var req struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Price       float64 `json:"price"`
			Category    string  `json:"category"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var product models.Product
		err := database.DB.
			Where("id = ?", productID).
			First(&product).Error

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}

		product.Name = req.Name
		product.Description = req.Description
		product.Price = req.Price
		product.Category = req.Category

		err = database.DB.Save(product).Error
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Product is updated")
		return

	case http.MethodDelete:
		if idStr == "" || idStr == "/" {
			http.Error(w, "Product ID is missing", http.StatusBadRequest)
			return

		}

		var product models.Product

		err := database.DB.
			Where("id = ?", productID).
			Delete(&product).Error

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Product is deleted")

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

}

func SearchProductHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("q")
		category := r.URL.Query().Get("category")
		minPriceStr := r.URL.Query().Get("min_price")
		maxPriceStr := r.URL.Query().Get("max_price")
		sort := r.URL.Query().Get("sort")

		minPrice, _ := strconv.ParseFloat(minPriceStr, 64)
		maxPrice, _ := strconv.ParseFloat(maxPriceStr, 64)

		page, limit := getPaginationParams(r)

		// Enforce positive pagination
		if page <= 0 {
			page = 1
		}
		if limit <= 0 {
			limit = 10
		}

		query := database.DB.Model(&models.Product{})

		if q != "" {
			query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+q+"%")
		}

		if category != "" {
			query = query.Where("category = ?", category)
		}

		if minPrice > 0 {
			query = query.Where("price >= ?", minPrice)
		}

		if maxPrice > 0 {
			query = query.Where("price <= ?", maxPrice)
		}

		// Sorting (default newest)
		switch strings.ToLower(sort) {
		case "price":
			query = query.Order("price ASC")
		case "price_desc":
			query = query.Order("price DESC")
		case "name":
			query = query.Order("name ASC")
		case "name_desc":
			query = query.Order("name DESC")
		default:
			query = query.Order("created_at DESC")
		}

		// Pagination
		offset := (page - 1) * limit
		query = query.Limit(limit).Offset(offset)

		// Fetch Products
		var products []models.Product
		if err := query.Find(&products).Error; err != nil {
			http.Error(w, "failed to fetch products", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"page":     page,
			"limit":    limit,
			"count":    len(products),
			"products": products,
		})

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

}

func BulkUpdateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		type BulkRequest struct {
			Products []models.Product `json:"products"`
		}

		var req BulkRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if len(req.Products) == 0 {
			http.Error(w, "no products to update", http.StatusBadRequest)
			return
		}

		const workerCount = 10
		jobs := make(chan models.Product, len(req.Products))
		results := make(chan bool, len(req.Products))

		// Worker function
		for range workerCount {
			go func() {
				for p := range jobs {
					// Only update the fields provided (e.g. price)
					err := database.DB.Model(&models.Product{}).
						Where("id = ?", p.ID).
						Updates(map[string]interface{}{
							"price": p.Price,
						}).Error
					results <- err == nil
				}
			}()
		}

		// Send jobs
		for _, p := range req.Products {
			jobs <- p
		}
		close(jobs)

		// Collect results
		succeeded, failed := 0, 0
		for i := 0; i < len(req.Products); i++ {
			if <-results {
				succeeded++
			} else {
				failed++
			}
		}

		// Response summary
		resp := map[string]int{
			"total":     len(req.Products),
			"succeeded": succeeded,
			"failed":    failed,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}
