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

// XEntityGeneral is the nested "general" object returned by Xentral v1 for
// companies (customers, suppliers). Name and address live here.
type XEntityGeneral struct {
	Name    string   `json:"name"`
	Address XAddress `json:"address"`
}

// XCustomer represents a customer record from Xentral.
type XCustomer struct {
	ID             string         `json:"id"`
	CustomerNumber string         `json:"customerNumber"`
	General        XEntityGeneral `json:"general"` // primary: general.name / general.address
	Company        string         `json:"company"` // flat fallback (older API versions)
	Name           string         `json:"name"`    // flat fallback
	Firma          string         `json:"firma"`   // flat fallback
	FirstName      string         `json:"firstName"`
	LastName       string         `json:"lastName"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	Address        XAddress       `json:"address"` // flat fallback
}

// ResolvedCompany returns the first non-empty company/name field.
func (c *XCustomer) ResolvedCompany() string {
	for _, v := range []string{c.General.Name, c.Company, c.Name, c.Firma} {
		if v != "" {
			return v
		}
	}
	return strings.TrimSpace(c.FirstName + " " + c.LastName)
}

// ResolvedAddress returns the address, preferring the nested general.address.
func (c *XCustomer) ResolvedAddress() XAddress {
	if c.General.Address.Street != "" || c.General.Address.City != "" {
		return c.General.Address
	}
	return c.Address
}

// XSupplier represents a supplier record from Xentral.
type XSupplier struct {
	ID             string         `json:"id"`
	SupplierNumber string         `json:"supplierNumber"`
	General        XEntityGeneral `json:"general"` // primary: general.name / general.address
	Company        string         `json:"company"` // flat fallback (older API versions)
	Name           string         `json:"name"`    // flat fallback
	Firma          string         `json:"firma"`   // flat fallback
	FirstName      string         `json:"firstName"`
	LastName       string         `json:"lastName"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	Origin         string         `json:"origin"`
}

// ResolvedCompany returns the first non-empty company/name field.
func (s *XSupplier) ResolvedCompany() string {
	for _, v := range []string{s.General.Name, s.Company, s.Name, s.Firma} {
		if v != "" {
			return v
		}
	}
	return strings.TrimSpace(s.FirstName + " " + s.LastName)
}

// XPurchaseOrder represents a purchase order (Lieferantenbestellung) from Xentral v3.
// The actual API response uses documentDate for the date, address.id for the supplier
// reference, and totals.gross for the total amount. Legacy field names are kept as
// fallbacks for older API versions.
type XPurchaseOrder struct {
	ID             string        `json:"id"`
	DocumentNumber string        `json:"documentNumber"`
	OrderNumber    string        `json:"orderNumber"`   // fallback
	Status         string        `json:"status"`
	DocumentDate   string        `json:"documentDate"`  // primary date field (v3 response)
	Date           string        `json:"date"`          // fallback
	OrderDate      string        `json:"orderDate"`     // fallback
	Address        XPORef        `json:"address"`       // supplier ref: address.id = supplier id
	Supplier       XPOSupplier   `json:"supplier"`      // fallback for older API versions
	SupplierNumber string        `json:"supplierNumber"`
	Totals         XPOTotals     `json:"totals"`        // totals.gross.amount + currency
	TotalNet       float64       `json:"totalNet"`      // fallback flat field
	Currency       string        `json:"currency"`      // fallback flat field
	LineItems      []XPOLineItem `json:"lineItems"`     // populated via separate fetch
}

// XPOTotals holds the financial totals of a purchase order.
type XPOTotals struct {
	Gross XPOAmount `json:"gross"`
	Net   XPOAmount `json:"net"`
}

// XPOAmount is a monetary amount returned as a string + currency.
type XPOAmount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// ResolvedOrderNumber returns the first non-empty order/document number.
func (o *XPurchaseOrder) ResolvedOrderNumber() string {
	if o.DocumentNumber != "" {
		return o.DocumentNumber
	}
	return o.OrderNumber
}

// ResolvedDate returns the first non-empty date string across known field names.
func (o *XPurchaseOrder) ResolvedDate() string {
	for _, d := range []string{o.DocumentDate, o.Date, o.OrderDate} {
		if d != "" {
			return d
		}
	}
	return ""
}

// ResolvedSupplierID returns the supplier/address ID used to look up the local mapping.
// The v3 API stores the supplier as address.id; older versions use supplier.id.
func (o *XPurchaseOrder) ResolvedSupplierID() string {
	if o.Address.ID != "" {
		return o.Address.ID
	}
	return o.Supplier.ID
}

// ResolvedTotalNet returns the net total amount parsed from whichever field is present.
func (o *XPurchaseOrder) ResolvedTotalNet() float64 {
	for _, a := range []string{o.Totals.Net.Amount, o.Totals.Gross.Amount} {
		if a != "" {
			if f, err := strconv.ParseFloat(a, 64); err == nil {
				return f
			}
		}
	}
	return o.TotalNet
}

// ResolvedCurrency returns the currency from whichever field is populated.
func (o *XPurchaseOrder) ResolvedCurrency() string {
	for _, c := range []string{o.Totals.Net.Currency, o.Totals.Gross.Currency, o.Currency} {
		if c != "" {
			return c
		}
	}
	return "EUR"
}

// XWarehouse represents a Xentral warehouse (Lagerort).
type XWarehouse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortname"` // Xentral field: "shortname"
	Active    bool   `json:"active"`
}

// XStorageLocation represents a Xentral storage location (Lagerplatz) within a warehouse.
type XStorageLocation struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Warehouse   XEntityRef `json:"warehouse"`
	Aisle       string     `json:"aisle"`
	Rack        string     `json:"rack"`
	Level       string     `json:"level"`
	Active      bool       `json:"active"`
}

// XEntityRef is a slim reference returned inside nested objects.
type XEntityRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// -------------------------------------------------------------------
// Product Stocks (undocumented /api/v1/stocks)
// -------------------------------------------------------------------

// XProductStock represents one stock entry from the undocumented Xentral stocks endpoint.
// Field names follow the observed Xentral response format; zero values mean "not applicable".
type XProductStock struct {
	ID              string     `json:"id"`
	Article         XEntityRef `json:"article"`         // {id, name/articleNumber}
	Warehouse       XEntityRef `json:"warehouse"`
	StorageLocation XEntityRef `json:"storageLocation"` // Lagerplatz; may be empty
	Quantity        float64    `json:"quantity"`
	ReservedQty     float64    `json:"reservedQuantity"`
	Unit            string     `json:"unit"`
	// Tracking (optional fields — not every installation / article uses all of them)
	BestBefore   string `json:"bestBefore"`   // MHD as "YYYY-MM-DD" or empty
	SerialNumber string `json:"serialNumber"` // Seriennummer
	BatchNumber  string `json:"batchNumber"`  // Charge
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
	Postcode string `json:"postcode"` // used by some Xentral versions
	Zip      string `json:"zip"`      // used by others (e.g. general.address.zip)
	Country  string `json:"country"`
}

// ResolvedZip returns the first non-empty postcode/zip field.
func (a *XAddress) ResolvedZip() string {
	if a.Zip != "" {
		return a.Zip
	}
	return a.Postcode
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
// Xentral v1 endpoints accept page[size] between 10 and 50.
// v3 cursor endpoints tolerate higher values but we keep one constant for simplicity.
const pageSize = 50

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
// Xentral using cursor pagination (x-pagination header). LineItems are included
// in the list response by the v3 API.
func (c *Client) ListPurchaseOrders(ctx context.Context) ([]XPurchaseOrder, error) {
	return listAllCursor[XPurchaseOrder](c, ctx, "/api/v3/purchaseOrders")
}

// ListPurchaseOrderLineItems fetches all line items for a single purchase order.
// The v3 list endpoint does not include line items inline; they must be fetched separately.
// Endpoint: GET /api/v3/purchaseOrders/{id}/lineItems
func (c *Client) ListPurchaseOrderLineItems(ctx context.Context, orderID string) ([]XPOLineItem, error) {
	return listAllCursor[XPOLineItem](c, ctx, "/api/v3/purchaseOrders/"+orderID+"/lineItems")
}

// ListSalesOrders fetches all sales orders (Verkaufsaufträge) from Xentral
// using cursor pagination (x-pagination header).
// The v3 salesOrders endpoint requires the feature flag "api-v3-sales-orders" on the instance.
func (c *Client) ListSalesOrders(ctx context.Context) ([]XSalesOrder, error) {
	return listAllCursor[XSalesOrder](c, ctx, "/api/v3/salesOrders")
}

// ListWarehouses fetches all warehouses (Lagerorte) from Xentral.
func (c *Client) ListWarehouses(ctx context.Context) ([]XWarehouse, error) {
	var all []XWarehouse
	for page := 1; ; page++ {
		batch, err := listPage[XWarehouse](c, ctx, "/api/v1/warehouses", page)
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

// ListStorageLocations fetches all storage locations (Lagerplätze) from Xentral.
func (c *Client) ListStorageLocations(ctx context.Context) ([]XStorageLocation, error) {
	var all []XStorageLocation
	for page := 1; ; page++ {
		batch, err := listPage[XStorageLocation](c, ctx, "/api/v1/storageLocations", page)
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

// ListProductStocks fetches current stock levels from the undocumented /api/v1/stocks endpoint.
func (c *Client) ListProductStocks(ctx context.Context) ([]XProductStock, error) {
	var all []XProductStock
	for page := 1; ; page++ {
		batch, err := listPage[XProductStock](c, ctx, "/api/v1/stocks", page)
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

// do executes a rate-limited GET with 429-retry.
// extraHeaders are added to the outgoing request.
// Returns the response body and headers.
func (c *Client) do(ctx context.Context, path string, params url.Values, extraHeaders map[string]string) ([]byte, http.Header, error) {
	const maxRetries = 3
	for attempt := 0; ; attempt++ {
		c.mu.Lock()
		if wait := c.minInterval - time.Since(c.lastReqAt); wait > 0 {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
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
			return nil, nil, fmt.Errorf("xentral: build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("xentral: request %s: %w", path, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, nil, fmt.Errorf("xentral: read body: %w", readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return nil, nil, fmt.Errorf("xentral: unauthorized – check API token")
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt >= maxRetries {
				return nil, nil, fmt.Errorf("xentral: rate limit reached (after %d retries)", maxRetries)
			}
			retryAfter := 60 * time.Second
			if s := resp.Header.Get("Retry-After"); s != "" {
				if secs, err := strconv.Atoi(s); err == nil && secs > 0 {
					retryAfter = time.Duration(secs) * time.Second
				}
			}
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(retryAfter):
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, nil, fmt.Errorf("xentral: HTTP %d from %s: %s", resp.StatusCode, path, string(body))
		}
		return body, resp.Header, nil
	}
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	body, _, err := c.do(ctx, path, params, nil)
	return body, err
}

// getCursor sends cursor in the x-pagination request header and returns
// the response body plus the next cursor from the x-pagination response header.
func (c *Client) getCursor(ctx context.Context, path string, params url.Values, cursor string) ([]byte, string, error) {
	var hdrs map[string]string
	if cursor != "" {
		hdrs = map[string]string{"x-pagination": cursor}
	}
	body, respHdr, err := c.do(ctx, path, params, hdrs)
	if err != nil {
		return nil, "", err
	}
	return body, respHdr.Get("x-pagination"), nil
}

// listAllCursor fetches all pages of a v3 resource using cursor pagination
// (x-pagination request/response header).
func listAllCursor[T any](c *Client, ctx context.Context, path string) ([]T, error) {
	params := url.Values{"page[size]": []string{fmt.Sprintf("%d", pageSize)}}
	var all []T
	cursor := ""
	for {
		body, nextCursor, err := c.getCursor(ctx, path, params, cursor)
		if err != nil {
			return nil, err
		}
		var result xList[T]
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("xentral: decode %s: %w", path, err)
		}
		all = append(all, result.Data...)
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}
	return all, nil
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
// The "price" field varies across Xentral versions: it can be a plain float64
// or a nested object {"amount": "10.50", "currency": "EUR"}.
type XSalesPrice struct {
	ID           string          `json:"id"`
	Article      XPriceProduct   `json:"article"`
	Name         string          `json:"name"` // price group / list name
	Price        json.RawMessage `json:"price"`
	Currency     string          `json:"currency"`
	FromQuantity float64         `json:"fromQuantity"`
	ValidFrom    string          `json:"validFrom"`
	ValidTo      string          `json:"validTo"`
	// Some Xentral versions expose the amount at the top level:
	PriceAmount json.Number `json:"amount"` // fallback field
}

// ResolvedPrice extracts the numeric price regardless of how Xentral encodes it.
// Handles: plain float64, {"amount": ..., "currency": ...}, top-level "amount".
func (p *XSalesPrice) ResolvedPrice() float64 {
	if len(p.Price) > 0 {
		var f float64
		if err := json.Unmarshal(p.Price, &f); err == nil {
			return f
		}
		var obj struct {
			Amount json.Number `json:"amount"`
		}
		if err := json.Unmarshal(p.Price, &obj); err == nil {
			if f, err := obj.Amount.Float64(); err == nil {
				return f
			}
		}
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
