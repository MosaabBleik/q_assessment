package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MosaabBleik/products-service/internal/models"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	defaultPage  = 1
	defaultLimit = 10
)

type ProductResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

func getPaginationParams(r *http.Request) (page int, limit int) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Init with default values
	page = defaultPage
	limit = defaultLimit

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			// Enfore positive value
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

type ProductHandler struct {
	DB          *gorm.DB
	RedisClient *redis.Client
}

func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	page, limit := getPaginationParams(r)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	var products []models.Product
	offset := (page - 1) * limit
	if err := h.DB.Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		http.Error(w, "Failed to fetch products", http.StatusInternalServerError)
		return
	}

	response := make([]ProductResponse, 0)
	for _, p := range products {
		response = append(response, ProductResponse{
			ID:       p.ID,
			Name:     p.Name,
			Price:    p.Price,
			Category: p.Category,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"page":     page,
		"limit":    limit,
		"count":    len(response),
		"products": response,
	})
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	id := vars["id"]

	var product models.Product
	if err := h.DB.Where("id = ?", id).First(&product).Error; err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	product := models.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}

	if err := h.DB.Create(&product).Error; err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := context.Background()

	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Category    string  `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var product models.Product
	if err := h.DB.Where("id = ?", id).First(&product).Error; err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	product.Name = req.Name
	product.Description = req.Description
	product.Price = req.Price
	product.Category = req.Category

	if err := h.DB.Save(&product).Error; err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	h.RedisClient.FlushDB(ctx)

	json.NewEncoder(w).Encode(product)
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.DB.Delete(&models.Product{}, "id = ?", id).Error; err != nil {
		http.Error(w, "Failed to delete product", http.StatusInternalServerError)
		return
	}

	h.RedisClient.FlushDB(ctx)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Product deleted successfully"))
}

func (h *ProductHandler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := context.Background()

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

	// --- Build Redis cache key ---
	cacheKey := fmt.Sprintf(
		"products:search:q=%s:cat=%s:min=%.2f:max=%.2f:sort=%s:page=%d:limit=%d",
		q, category, minPrice, maxPrice, sort, page, limit,
	)

	// --- Try to get cached result ---
	cached, err := h.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		w.Write([]byte(cached)) // cache hit
		return
	}

	query := h.DB.Model(&models.Product{})
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

	response := make([]ProductResponse, 0)
	for _, p := range products {
		response = append(response, ProductResponse{
			ID:       p.ID,
			Name:     p.Name,
			Price:    p.Price,
			Category: p.Category,
		})
	}

	result := map[string]any{
		"page":     page,
		"limit":    limit,
		"count":    len(response),
		"products": response,
	}

	// --- Marshal to JSON ---
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	// --- Save to Redis with TTL ---
	_ = h.RedisClient.Set(ctx, cacheKey, jsonBytes, 5*time.Minute).Err()

	// --- Return response ---
	w.Write(jsonBytes)
}

func (h *ProductHandler) BulkUpdate(w http.ResponseWriter, r *http.Request) {

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
				err := h.DB.Model(&models.Product{}).
					Where("id = ?", p.ID).
					Updates(map[string]any{
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
}

func (h *ProductHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	dbStatus := "ok"
	redisStatus := "ok"

	if err := h.DB.Exec("SELECT 1").Error; err != nil {
		dbStatus = "error"
	}

	if err := h.RedisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "error"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"database": dbStatus,
		"redis":    redisStatus,
	})
}
