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
//   GET /api/v3/purchaseOrders  – purchase orders (Lieferantenbestellungen)
package xentral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

// XPurchaseOrder represents a purchase order (Lieferantenbestellung) from Xentral v3.
type XPurchaseOrder struct {
	ID             string              `json:"id"`
	OrderNumber    string              `json:"orderNumber"`
	DocumentNumber string              `json:"documentNumber"`
	Status         string              `json:"status"`
	Date           string              `json:"date"`
	OrderDate      string              `json:"orderDate"`      // fallback field name
	Supplier       XPOSupplier         `json:"supplier"`
	LineItems      []XPOLineItem       `json:"lineItems"`
	TotalNet       float64             `json:"totalNet"`
	Currency       string              `json:"currency"`
}

// ResolvedOrderNumber returns the first non-empty order/document number.
func (o *XPurchaseOrder) ResolvedOrderNumber() string {
	if o.OrderNumber != "" {
		return o.OrderNumber
	}
	return o.DocumentNumber
}

// ResolvedDate returns the first non-empty date string.
func (o *XPurchaseOrder) ResolvedDate() string {
	if o.Date != "" {
		return o.Date
	}
	return o.OrderDate
}

type XPOSupplier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type XPOLineItem struct {
	ID          string     `json:"id"`
	Article     XPOArticle `json:"article"`
	Description string     `json:"description"`
	Quantity    float64    `json:"quantity"`
	UnitPrice   float64    `json:"unitPrice"`
}

type XPOArticle struct {
	ID            string `json:"id"`
	ArticleNumber string `json:"articleNumber"`
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

// ListPurchaseOrders fetches all purchase orders (Lieferantenbestellungen) from
// Xentral across all pages using the v3 API.
func (c *Client) ListPurchaseOrders(ctx context.Context) ([]XPurchaseOrder, error) {
	var all []XPurchaseOrder
	for page := 1; ; page++ {
		batch, err := listPage[XPurchaseOrder](c, ctx, "/api/v3/purchaseOrders", page)
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

// ListSalesOrders fetches all sales orders (Verkaufsaufträge) from Xentral
// across all pages using the v3 API.
// The v3 salesOrders endpoint requires the feature flag "api-v3-sales-orders" on the instance.
func (c *Client) ListSalesOrders(ctx context.Context) ([]XSalesOrder, error) {
	var all []XSalesOrder
	for page := 1; ; page++ {
		batch, err := listPage[XSalesOrder](c, ctx, "/api/v3/salesOrders", page)
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

// PatchProductEAN pushes an EAN barcode back to the matching Xentral article.
// Uses PATCH /api/v1/products/{xentralArticleID}.
// This is called after we allocate a new EAN from an EANRange so Xentral stays in sync.
func (c *Client) PatchProductEAN(ctx context.Context, xentralArticleID, ean string) error {
	body := map[string]string{"ean": ean}
	return c.patch(ctx, "/api/v1/products/"+xentralArticleID, body)
}

// PatchProductFreefields pushes freefield values (freifeld1…freifeld10) back to a Xentral article.
// freefields is a map from Xentral key (e.g. "freifeld1") to value string.
func (c *Client) PatchProductFreefields(ctx context.Context, xentralArticleID string, freefields map[string]string) error {
	payload := make(map[string]interface{}, len(freefields))
	for k, v := range freefields {
		payload[k] = v
	}
	return c.patch(ctx, "/api/v1/products/"+xentralArticleID, payload)
}

// -------------------------------------------------------------------
// HTTP helper
// -------------------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	const maxRetries = 3
	for attempt := 0; ; attempt++ {
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
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("xentral: read body: %w", readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("xentral: unauthorized – check API token")
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt >= maxRetries {
				return nil, fmt.Errorf("xentral: rate limit reached (after %d retries)", maxRetries)
			}
			// Respect Retry-After header; default to 60 s.
			retryAfter := 60 * time.Second
			if s := resp.Header.Get("Retry-After"); s != "" {
				if secs, err := strconv.Atoi(s); err == nil && secs > 0 {
					retryAfter = time.Duration(secs) * time.Second
				}
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("xentral: HTTP %d from %s: %s", resp.StatusCode, path, string(body))
		}
		return body, nil
	}
}

// CreatePurchaseOrder creates a new purchase order in Xentral and returns the created ID.
// Endpoint: POST /api/v3/purchaseOrders
func (c *Client) CreatePurchaseOrder(ctx context.Context, req XPOCreateRequest) (string, error) {
	body, err := c.postJSON(ctx, "/api/v3/purchaseOrders", req)
	if err != nil {
		return "", err
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		ID string `json:"id"` // some versions return flat
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("xentral: parse create PO response: %w", err)
	}
	if resp.Data.ID != "" {
		return resp.Data.ID, nil
	}
	return resp.ID, nil
}

// PatchPurchaseOrder updates an existing purchase order header in Xentral.
// Endpoint: PATCH /api/v3/purchaseOrders/{id}
func (c *Client) PatchPurchaseOrder(ctx context.Context, xentralID string, req XPOPatchRequest) error {
	return c.patch(ctx, "/api/v3/purchaseOrders/"+xentralID, req)
}

// CreatePurchaseOrderLineItem adds a line item to an existing Xentral purchase order.
// Endpoint: POST /api/v3/purchaseOrders/{id}/lineItems
func (c *Client) CreatePurchaseOrderLineItem(ctx context.Context, purchaseOrderID string, item XPOLineItemCreate) error {
	_, err := c.postJSON(ctx, "/api/v3/purchaseOrders/"+purchaseOrderID+"/lineItems", item)
	return err
}

// -------------------------------------------------------------------
// Purchase order write types
// -------------------------------------------------------------------

// XPOCreateRequest is the body for POST /api/v3/purchaseOrders.
type XPOCreateRequest struct {
	Supplier  XPORef              `json:"supplier"`
	Date      string              `json:"date,omitempty"`      // "YYYY-MM-DD"
	Notes     string              `json:"notes,omitempty"`
	LineItems []XPOLineItemCreate `json:"lineItems,omitempty"` // included inline if API supports it
}

// XPOPatchRequest is the body for PATCH /api/v3/purchaseOrders/{id}.
type XPOPatchRequest struct {
	Date  string `json:"date,omitempty"`
	Notes string `json:"notes,omitempty"`
}

// XPORef is a reference to a Xentral entity by ID.
type XPORef struct {
	ID string `json:"id"`
}

// XPOLineItemCreate is a line item for POST /api/v3/purchaseOrders/{id}/lineItems.
type XPOLineItemCreate struct {
	Article     XPORef  `json:"article"`
	Description string  `json:"description,omitempty"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice,omitempty"`
}

// ListPurchasePrices fetches all purchase price entries (EK-Preislisten) from Xentral.
// Endpoint: GET /api/v1/purchasePrices
func (c *Client) ListPurchasePrices(ctx context.Context) ([]XPurchasePrice, error) {
	var all []XPurchasePrice
	for page := 1; ; page++ {
		batch, err := listPage[XPurchasePrice](c, ctx, "/api/v1/purchasePrices", page)
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

// ListSalesPrices fetches all sales price entries (VK-Preislisten) from Xentral.
// Endpoint: GET /api/v3/salesPrices
// Requires feature flag "api-v3-sales-prices" on the Xentral instance.
func (c *Client) ListSalesPrices(ctx context.Context) ([]XSalesPrice, error) {
	var all []XSalesPrice
	for page := 1; ; page++ {
		batch, err := listPage[XSalesPrice](c, ctx, "/api/v3/salesPrices", page)
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
// Purchase/Sales price types
// -------------------------------------------------------------------

// XPurchasePrice is a purchase price entry from Xentral GET /api/v1/purchasePrices.
type XPurchasePrice struct {
	ID           string         `json:"id"`
	Product      XPriceProduct  `json:"product"`
	Supplier     XPriceSupplier `json:"supplier"`
	Price        float64        `json:"price"`
	Currency     string         `json:"currency"`
	FromQuantity float64        `json:"fromQuantity"`
	ValidFrom    string         `json:"validFrom"`
	ExpiresAt    string         `json:"expiresAt"`
	Name         string         `json:"name"` // price list name if provided
}

// XSalesPrice is a sales price entry from Xentral GET /api/v3/salesPrices.
type XSalesPrice struct {
	ID             string        `json:"id"`
	Article        XPriceProduct `json:"article"`
	Name           string        `json:"name"`           // price group / list name
	Price          float64       `json:"price"`
	Currency       string        `json:"currency"`
	FromQuantity   float64       `json:"fromQuantity"`
	ValidFrom      string        `json:"validFrom"`
	ValidTo        string        `json:"validTo"`
	// Some Xentral versions nest the price differently:
	PriceAmount    json.Number   `json:"amount"`         // fallback field
}

// ResolvedPrice returns the price, checking the nested amount field as a fallback.
func (p *XSalesPrice) ResolvedPrice() float64 {
	if p.Price != 0 {
		return p.Price
	}
	f, _ := p.PriceAmount.Float64()
	return f
}

type XPriceProduct struct {
	ID            string `json:"id"`
	ArticleNumber string `json:"articleNumber"`
}

type XPriceSupplier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) postJSON(ctx context.Context, path string, payload interface{}) ([]byte, error) {
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

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("xentral: marshal post body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("xentral: build post request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xentral: post %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

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

func (c *Client) patch(ctx context.Context, path string, payload interface{}) error {
	c.mu.Lock()
	if wait := c.minInterval - time.Since(c.lastReqAt); wait > 0 {
		c.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		c.mu.Lock()
	}
	c.lastReqAt = time.Now()
	c.mu.Unlock()

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("xentral: marshal patch body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("xentral: build patch request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("xentral: patch %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("xentral: unauthorized – check API token")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("xentral: rate limit reached")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("xentral: HTTP %d from %s: %s", resp.StatusCode, path, string(body))
	}
	return nil
}

// -------------------------------------------------------------------
// Sales Order types (v3)
// -------------------------------------------------------------------

// XSalesOrder represents an outbound sales order (Verkaufsauftrag) from Xentral v3.
// The v3 endpoint requires feature flag "api-v3-sales-orders" on the instance.
type XSalesOrder struct {
	ID                  string          `json:"id"`
	DocumentNumber      string          `json:"documentNumber"`
	ExternalOrderNumber string          `json:"externalOrderNumber"`
	Date                string          `json:"date"`
	Status              string          `json:"status"`
	Customer            XSOCustomer     `json:"customer"`
	LineItems           []XSOLineItem   `json:"lineItems"`
	NetSales            XSOAmount       `json:"netSales"`
	Total               XSOAmount       `json:"total"`
	Currency            string          `json:"currency"`
}

type XSOCustomer struct {
	ID     string `json:"id"`
	Number string `json:"number"`
}

// XSOLineItem is one line in a Xentral v3 sales order.
type XSOLineItem struct {
	ID          string     `json:"id"`
	Article     XSOArticle `json:"article"`
	Description string     `json:"description"`
	Quantity    float64    `json:"quantity"`
	UnitPrice   float64    `json:"unitPrice"`
}

type XSOArticle struct {
	ID            string `json:"id"`
	ArticleNumber string `json:"articleNumber"`
}

// XSOAmount is a monetary value as returned by the v3 sales orders API.
// Xentral returns amount as a string in some versions.
type XSOAmount struct {
	Amount   json.Number `json:"amount"`
	Currency string      `json:"currency"`
}

// Float64 converts the amount to float64, returning 0 on parse error.
func (a XSOAmount) Float64() float64 {
	f, _ := a.Amount.Float64()
	return f
}
