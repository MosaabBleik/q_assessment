package product_clients

import (
	// "encoding/json"
	// "fmt"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	// "strconv"
	// "strings"
)

type ProductsClient struct {
	baseURL    string
	httpClient *http.Client
}

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
}

func NewProductsClient(baseURL string) *ProductsClient {
	return &ProductsClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *ProductsClient) GetProduct(ctx context.Context, productID string) (*Product, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/products/%s", c.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach products service: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var product Product
		if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
			return nil, fmt.Errorf("failed to decode product: %v", err)
		}
		return &product, nil

	case http.StatusNotFound:
		return nil, fmt.Errorf("product not found (id=%s)", productID)

	case http.StatusGatewayTimeout, http.StatusServiceUnavailable:
		return nil, fmt.Errorf("products service unavailable")

	default:
		return nil, fmt.Errorf("unexpected response from products service: %s", resp.Status)
	}
}
