package product_clients

import (
	// "encoding/json"
	// "fmt"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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

func NewProductsClient(baseURL string, seconds int) *ProductsClient {
	return &ProductsClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: time.Duration(seconds) * time.Second},
	}
}

func (c *ProductsClient) GetProduct(ctx context.Context, productID string) (*Product, int, error) {
	url := fmt.Sprintf("%s/api/products/%s", c.baseURL, productID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 1, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, 2, fmt.Errorf("request timed out: %w", err)
		}
		if errors.Is(err, context.Canceled) {
			return nil, 3, fmt.Errorf("request canceled: %w", err)
		}

		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return nil, 4, fmt.Errorf("network timeout: %w", err)
			}
			return nil, 5, fmt.Errorf("network error: %w", err)
		}

		return nil, 6, fmt.Errorf("failed to reach products service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var product Product
		if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
			return nil, 3, fmt.Errorf("failed to decode product: %v", err)
		}
		return &product, 0, nil

	case http.StatusNotFound:
		return nil, 4, fmt.Errorf("product not found (id=%s)", productID)

	case http.StatusGatewayTimeout, http.StatusServiceUnavailable:
		return nil, 5, fmt.Errorf("products service unavailable")

	default:
		return nil, 6, fmt.Errorf("unexpected response from products service: %s", resp.Status)
	}
}
