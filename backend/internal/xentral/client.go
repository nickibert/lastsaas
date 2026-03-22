// Package xentral provides an HTTP client and sync engine for the Xentral ERP REST API.
//
// Xentral API reference: https://github.com/xentral/api-spec-public
// Authentication: Bearer token passed in the Authorization header.
// Base URL pattern: https://{xentralId}.xentral.biz
//
// Endpoint paths used:
//   GET /api/v1/products        – article list
//   GET /api/v1/customers       – customer list
//   GET /api/v1/suppliers       – supplier list
//   GET /api/v1/sales-orders    – incoming sales orders
package xentral

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a thin HTTP wrapper around the Xentral REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a Xentral API client.
// baseURL should be e.g. "https://mycompany.xentral.biz" (no trailing slash).
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Ping calls a lightweight endpoint to verify the connection and credentials.
func (c *Client) Ping(ctx context.Context) error {
	// Use the products endpoint with limit=1 as a health check
	_, err := c.get(ctx, "/api/v1/products", url.Values{"limit": []string{"1"}})
	return err
}

// -------------------------------------------------------------------
// Xentral data types
// -------------------------------------------------------------------

// XArticle represents a product/article record from Xentral.
type XArticle struct {
	ID            string  `json:"id"`
	ArticleNumber string  `json:"articleNumber"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	EAN           string  `json:"ean"`
	PurchasePrice float64 `json:"purchasePrice"`
	SellPrice     float64 `json:"sellPrice"`
	Weight        float64 `json:"weight"`
	Active        bool    `json:"active"`
}

// XCustomer represents a customer record from Xentral.
type XCustomer struct {
	ID             string         `json:"id"`
	CustomerNumber string         `json:"customerNumber"`
	Company        string         `json:"company"`
	FirstName      string         `json:"firstName"`
	LastName       string         `json:"lastName"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	Address        XAddress       `json:"address"`
}

// XSupplier represents a supplier record from Xentral.
type XSupplier struct {
	ID             string   `json:"id"`
	SupplierNumber string   `json:"supplierNumber"`
	Company        string   `json:"company"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	Email          string   `json:"email"`
	Phone          string   `json:"phone"`
	Origin         string   `json:"origin"`
}

// XSalesOrder represents an incoming sales order from Xentral.
type XSalesOrder struct {
	ID          string           `json:"id"`
	OrderNumber string           `json:"orderNumber"`
	Status      string           `json:"status"`
	OrderDate   string           `json:"orderDate"`
	Customer    XOrderCustomer   `json:"customer"`
	Positions   []XOrderPosition `json:"positions"`
	TotalNet    float64          `json:"totalNet"`
	Currency    string           `json:"currency"`
}

type XAddress struct {
	Street   string `json:"street"`
	City     string `json:"city"`
	Postcode string `json:"postcode"`
	Country  string `json:"country"`
}

type XOrderCustomer struct {
	ID      string `json:"id"`
	Company string `json:"company"`
}

type XOrderPosition struct {
	ProductID   string  `json:"productId"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
}

// paginated is the generic wrapper Xentral uses for list responses.
type paginated[T any] struct {
	Data []T    `json:"data"`
	Meta xMeta  `json:"meta"`
}

type xMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// -------------------------------------------------------------------
// Paginated list fetchers
// -------------------------------------------------------------------

const pageSize = 100

// ListArticles fetches all articles from Xentral across all pages.
func (c *Client) ListArticles(ctx context.Context) ([]XArticle, error) {
	var all []XArticle
	for page := 1; ; page++ {
		body, err := c.get(ctx, "/api/v1/products", url.Values{
			"page":  []string{fmt.Sprintf("%d", page)},
			"limit": []string{fmt.Sprintf("%d", pageSize)},
		})
		if err != nil {
			return nil, err
		}
		var result paginated[XArticle]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("xentral: decode articles page %d: %w", page, err)
		}
		all = append(all, result.Data...)
		if len(all) >= result.Meta.Total || len(result.Data) == 0 {
			break
		}
	}
	return all, nil
}

// ListCustomers fetches all customers from Xentral across all pages.
func (c *Client) ListCustomers(ctx context.Context) ([]XCustomer, error) {
	var all []XCustomer
	for page := 1; ; page++ {
		body, err := c.get(ctx, "/api/v1/customers", url.Values{
			"page":  []string{fmt.Sprintf("%d", page)},
			"limit": []string{fmt.Sprintf("%d", pageSize)},
		})
		if err != nil {
			return nil, err
		}
		var result paginated[XCustomer]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("xentral: decode customers page %d: %w", page, err)
		}
		all = append(all, result.Data...)
		if len(all) >= result.Meta.Total || len(result.Data) == 0 {
			break
		}
	}
	return all, nil
}

// ListSuppliers fetches all suppliers from Xentral across all pages.
func (c *Client) ListSuppliers(ctx context.Context) ([]XSupplier, error) {
	var all []XSupplier
	for page := 1; ; page++ {
		body, err := c.get(ctx, "/api/v1/suppliers", url.Values{
			"page":  []string{fmt.Sprintf("%d", page)},
			"limit": []string{fmt.Sprintf("%d", pageSize)},
		})
		if err != nil {
			return nil, err
		}
		var result paginated[XSupplier]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("xentral: decode suppliers page %d: %w", page, err)
		}
		all = append(all, result.Data...)
		if len(all) >= result.Meta.Total || len(result.Data) == 0 {
			break
		}
	}
	return all, nil
}

// ListSalesOrders fetches all sales orders from Xentral across all pages.
func (c *Client) ListSalesOrders(ctx context.Context) ([]XSalesOrder, error) {
	var all []XSalesOrder
	for page := 1; ; page++ {
		body, err := c.get(ctx, "/api/v1/sales-orders", url.Values{
			"page":  []string{fmt.Sprintf("%d", page)},
			"limit": []string{fmt.Sprintf("%d", pageSize)},
		})
		if err != nil {
			return nil, err
		}
		var result paginated[XSalesOrder]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("xentral: decode sales-orders page %d: %w", page, err)
		}
		all = append(all, result.Data...)
		if len(all) >= result.Meta.Total || len(result.Data) == 0 {
			break
		}
	}
	return all, nil
}

// -------------------------------------------------------------------
// HTTP helper
// -------------------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("xentral: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xentral: request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("xentral: read body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("xentral: unauthorized – check API token")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("xentral: rate limit reached")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("xentral: HTTP %d from %s: %s", resp.StatusCode, path, string(body))
	}
	return body, nil
}
