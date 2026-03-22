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
//   GET /api/v1/salesOrders    – incoming sales orders
package xentral

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is a thin HTTP wrapper around the Xentral REST API.
// It enforces a minimum interval between requests to stay within Xentral's
// default rate limit of 100 calls/minute (~600 ms per request → ~100/min).
type Client struct {
	baseURL     string
	token       string
	httpClient  *http.Client
	mu          sync.Mutex
	lastReqAt   time.Time
	minInterval time.Duration
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
		minInterval: 650 * time.Millisecond, // ~92 req/min, safely below 100/min limit
	}
}

// Ping calls a lightweight endpoint to verify the connection and credentials.
func (c *Client) Ping(ctx context.Context) error {
	// Use the products endpoint as a health check.
	// Xentral requires page[size] to be between 10 and 150 (string-encoded integer).
	_, err := c.get(ctx, "/api/v1/products", url.Values{
		"page[number]": []string{"1"},
		"page[size]":   []string{"10"},
	})
	return err
}

// GetSettings calls the undocumented /api/settings endpoint and returns
// the Xentral instance information (company name, version, address, …).
// The response schema is not officially documented so we capture common
// fields and store the raw JSON for forward-compatibility.
func (c *Client) GetSettings(ctx context.Context) (*XentralSettings, error) {
	body, err := c.get(ctx, "/api/settings", nil)
	if err != nil {
		return nil, fmt.Errorf("xentral: get settings: %w", err)
	}
	var s XentralSettings
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("xentral: decode settings: %w", err)
	}
	return &s, nil
}

// -------------------------------------------------------------------
// Xentral data types
// -------------------------------------------------------------------

// XentralSettings represents the response from the undocumented /api/settings endpoint.
// Fields are best-effort; the endpoint is not officially documented.
type XentralSettings struct {
	// Company / instance identity
	CompanyName string `json:"companyName"`
	// Alternate field names Xentral versions have used:
	Firma    string `json:"firma"`
	Name     string `json:"name"`
	ShopName string `json:"shopName"`

	// Version / edition
	Version string `json:"version"`
	Edition string `json:"edition"`

	// Contact
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Website  string `json:"website"`
	Language string `json:"language"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`

	// Tax
	TaxID string `json:"taxId"`
	VATID string `json:"vatId"`

	// Address (flat or nested – we try both)
	Street  string `json:"street"`
	ZIP     string `json:"zip"`
	City    string `json:"city"`
	Country string `json:"country"`
}

// ResolvedCompanyName returns the first non-empty company name field.
func (s *XentralSettings) ResolvedCompanyName() string {
	for _, v := range []string{s.CompanyName, s.Firma, s.Name, s.ShopName} {
		if v != "" {
			return v
		}
	}
	return ""
}

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
// Xentral uses "documentNumber" in v1/salesOrders responses; "orderNumber" is
// kept as a fallback for older API variants.
type XSalesOrder struct {
	ID             string           `json:"id"`
	OrderNumber    string           `json:"orderNumber"`
	DocumentNumber string           `json:"documentNumber"`
	Status         string           `json:"status"`
	OrderDate      string           `json:"orderDate"`
	DocumentDate   string           `json:"documentDate"`
	Customer       XOrderCustomer   `json:"customer"`
	Positions      []XOrderPosition `json:"positions"`
	TotalNet       float64          `json:"totalNet"`
	Currency       string           `json:"currency"`
}

// ResolvedOrderNumber returns the first non-empty order/document number.
func (o *XSalesOrder) ResolvedOrderNumber() string {
	if o.OrderNumber != "" {
		return o.OrderNumber
	}
	return o.DocumentNumber
}

// ResolvedOrderDate returns the first non-empty order/document date string.
func (o *XSalesOrder) ResolvedOrderDate() string {
	if o.OrderDate != "" {
		return o.OrderDate
	}
	return o.DocumentDate
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

// xentral wraps list responses – the data is always in "data".
// We intentionally ignore the meta/pagination object because its field names
// vary across Xentral versions; instead we stop when a page returns fewer
// items than requested (standard "last page" heuristic).
type xList[T any] struct {
	Data []T `json:"data"`
}

// -------------------------------------------------------------------
// Paginated list fetchers
// -------------------------------------------------------------------

// pageSize is the number of records requested per page.
// Xentral uses bracket-style pagination: page[number] and page[size].
const pageSize = 100

// listPage is the shared helper for fetching one page of any resource.
func listPage[T any](c *Client, ctx context.Context, path string, pageNum int) ([]T, error) {
	body, err := c.get(ctx, path, url.Values{
		"page[number]": []string{fmt.Sprintf("%d", pageNum)},
		"page[size]":   []string{fmt.Sprintf("%d", pageSize)},
	})
	if err != nil {
		return nil, err
	}
	var result xList[T]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("xentral: decode %s page %d: %w", path, pageNum, err)
	}
	return result.Data, nil
}

// ListArticles fetches all articles from Xentral across all pages.
func (c *Client) ListArticles(ctx context.Context) ([]XArticle, error) {
	var all []XArticle
	for page := 1; ; page++ {
		batch, err := listPage[XArticle](c, ctx, "/api/v1/products", page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

// ListCustomers fetches all customers from Xentral across all pages.
func (c *Client) ListCustomers(ctx context.Context) ([]XCustomer, error) {
	var all []XCustomer
	for page := 1; ; page++ {
		batch, err := listPage[XCustomer](c, ctx, "/api/v1/customers", page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

// ListSuppliers fetches all suppliers from Xentral across all pages.
func (c *Client) ListSuppliers(ctx context.Context) ([]XSupplier, error) {
	var all []XSupplier
	for page := 1; ; page++ {
		batch, err := listPage[XSupplier](c, ctx, "/api/v1/suppliers", page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

// ListSalesOrders fetches all sales orders from Xentral across all pages.
func (c *Client) ListSalesOrders(ctx context.Context) ([]XSalesOrder, error) {
	var all []XSalesOrder
	for page := 1; ; page++ {
		batch, err := listPage[XSalesOrder](c, ctx, "/api/v1/salesOrders", page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return all, nil
}

// -------------------------------------------------------------------
// HTTP helper
// -------------------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	// Enforce rate limit: wait until minInterval has passed since the last request.
	c.mu.Lock()
	if wait := c.minInterval - time.Since(c.lastReqAt); wait > 0 {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		c.mu.Lock()
	}
	c.lastReqAt = time.Now()
	c.mu.Unlock()

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
