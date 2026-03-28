package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lastsaas/internal/db"
	"lastsaas/internal/middleware"
	"lastsaas/internal/models"
	"lastsaas/internal/syslog"
	"lastsaas/internal/xentral"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/gorilla/mux"
)

type ProcurementHandler struct {
	db       *db.MongoDB
	syslog   *syslog.Logger
	tenantMW *middleware.TenantMiddleware
	xentral  *xentral.Engine
}

func NewProcurementHandler(database *db.MongoDB, sysLogger *syslog.Logger) *ProcurementHandler {
	return &ProcurementHandler{
		db:       database,
		syslog:   sysLogger,
		tenantMW: middleware.NewTenantMiddleware(database),
		xentral:  xentral.NewEngine(database, sysLogger),
	}
}

// maybePushOrder fires a real-time push of an order to Xentral in a goroutine.
// It is a no-op when the tenant has PushOrdersToXentral disabled or unconfigured.
func (h *ProcurementHandler) maybePushOrder(tenantID, orderID primitive.ObjectID) {
	go func() {
		ctx := context.Background()
		var cfg models.XentralConfig
		if err := h.db.XentralConfigs().FindOne(ctx, bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
			return
		}
		if !cfg.Enabled || !cfg.PushOrdersToXentral || cfg.BaseURL == "" || cfg.APIToken == "" {
			return
		}
		client := xentral.NewClient(cfg.BaseURL, cfg.APIToken)
		if _, err := h.xentral.PushOrderToXentral(ctx, tenantID, orderID, client); err != nil {
			h.syslog.Log(ctx, "medium", "xentral order push failed: "+err.Error())
		}
	}()
}

// RegisterRoutes wires all procurement endpoints onto the given router.
// All routes require authentication and a valid tenant context.
func (h *ProcurementHandler) RegisterRoutes(r *mux.Router, authMW mux.MiddlewareFunc) {
	s := r.PathPrefix("/api/procurement").Subrouter()
	s.Use(authMW)
	s.Use(h.tenantMW.RequireTenant)

	// Suppliers
	s.HandleFunc("/suppliers", h.listSuppliers).Methods(http.MethodGet)
	s.HandleFunc("/suppliers", h.createSupplier).Methods(http.MethodPost)
	s.HandleFunc("/suppliers/{id}", h.getSupplier).Methods(http.MethodGet)
	s.HandleFunc("/suppliers/{id}", h.updateSupplier).Methods(http.MethodPut)
	s.HandleFunc("/suppliers/{id}", h.deleteSupplier).Methods(http.MethodDelete)

	// Supplier codes
	s.HandleFunc("/supplier-codes", h.listSupplierCodes).Methods(http.MethodGet)
	s.HandleFunc("/supplier-codes", h.createSupplierCode).Methods(http.MethodPost)
	s.HandleFunc("/supplier-codes/{id}", h.updateSupplierCode).Methods(http.MethodPut)
	s.HandleFunc("/supplier-codes/{id}", h.deleteSupplierCode).Methods(http.MethodDelete)

	// Goods groups
	s.HandleFunc("/goods-groups", h.listGoodsGroups).Methods(http.MethodGet)
	s.HandleFunc("/goods-groups", h.createGoodsGroup).Methods(http.MethodPost)
	s.HandleFunc("/goods-groups/{id}", h.updateGoodsGroup).Methods(http.MethodPut)
	s.HandleFunc("/goods-groups/{id}", h.deleteGoodsGroup).Methods(http.MethodDelete)

	// Products
	s.HandleFunc("/products", h.listProducts).Methods(http.MethodGet)
	s.HandleFunc("/products", h.createProduct).Methods(http.MethodPost)
	s.HandleFunc("/products/{id}", h.getProduct).Methods(http.MethodGet)
	s.HandleFunc("/products/{id}", h.updateProduct).Methods(http.MethodPut)
	s.HandleFunc("/products/{id}", h.deleteProduct).Methods(http.MethodDelete)

	// Freight carriers
	s.HandleFunc("/freight-carriers", h.listFreightCarriers).Methods(http.MethodGet)
	s.HandleFunc("/freight-carriers", h.createFreightCarrier).Methods(http.MethodPost)
	s.HandleFunc("/freight-carriers/{id}", h.updateFreightCarrier).Methods(http.MethodPut)
	s.HandleFunc("/freight-carriers/{id}", h.deleteFreightCarrier).Methods(http.MethodDelete)

	// Harbours
	s.HandleFunc("/harbours", h.listHarbours).Methods(http.MethodGet)
	s.HandleFunc("/harbours", h.createHarbour).Methods(http.MethodPost)
	s.HandleFunc("/harbours/{id}", h.updateHarbour).Methods(http.MethodPut)
	s.HandleFunc("/harbours/{id}", h.deleteHarbour).Methods(http.MethodDelete)

	// Containers
	s.HandleFunc("/containers", h.listContainers).Methods(http.MethodGet)
	s.HandleFunc("/containers", h.createContainer).Methods(http.MethodPost)
	s.HandleFunc("/containers/{id}", h.updateContainer).Methods(http.MethodPut)
	s.HandleFunc("/containers/{id}", h.deleteContainer).Methods(http.MethodDelete)

	// Countries
	s.HandleFunc("/countries", h.listCountries).Methods(http.MethodGet)
	s.HandleFunc("/countries", h.createCountry).Methods(http.MethodPost)
	s.HandleFunc("/countries/seed", h.seedCountries).Methods(http.MethodPost)
	s.HandleFunc("/countries/{id}", h.updateCountry).Methods(http.MethodPut)
	s.HandleFunc("/countries/{id}", h.deleteCountry).Methods(http.MethodDelete)

	// Product price lists
	s.HandleFunc("/products/{id}/price-lists", h.listProductPriceLists).Methods(http.MethodGet)
	s.HandleFunc("/products/{id}/price-lists", h.createProductPriceList).Methods(http.MethodPost)
	s.HandleFunc("/products/{id}/price-lists/{priceListId}", h.updateProductPriceList).Methods(http.MethodPut)
	s.HandleFunc("/products/{id}/price-lists/{priceListId}", h.deleteProductPriceList).Methods(http.MethodDelete)

	// EK history + chart
	s.HandleFunc("/products/{id}/ek-history", h.getEKHistory).Methods(http.MethodGet)
	s.HandleFunc("/products/{id}/ek-history/chart", h.getEKHistoryChart).Methods(http.MethodGet)
	s.HandleFunc("/products/{id}/inventory-lots", h.listInventoryLots).Methods(http.MethodGet)
	s.HandleFunc("/products/{id}/inventory-valuation", h.getInventoryValuation).Methods(http.MethodGet)

	// Inventory import / export
	s.HandleFunc("/inventory/import", h.importInventoryLots).Methods(http.MethodPost)
	s.HandleFunc("/inventory/export", h.exportInventoryValuation).Methods(http.MethodGet)

	// Procurement tenant config
	s.HandleFunc("/config", h.getProcurementConfig).Methods(http.MethodGet)
	s.HandleFunc("/config", h.updateProcurementConfig).Methods(http.MethodPut)

	// Calendar
	s.HandleFunc("/calendar", h.getCalendar).Methods(http.MethodGet)

	// Orders
	s.HandleFunc("/orders", h.listOrders).Methods(http.MethodGet)
	s.HandleFunc("/orders", h.createOrder).Methods(http.MethodPost)
	s.HandleFunc("/orders/{id}", h.getOrder).Methods(http.MethodGet)
	s.HandleFunc("/orders/{id}", h.updateOrder).Methods(http.MethodPut)
	s.HandleFunc("/orders/{id}", h.deleteOrder).Methods(http.MethodDelete)
	s.HandleFunc("/orders/{id}/freight-date", h.patchOrderFreightDate).Methods(http.MethodPatch)

	// Order EK calculation
	s.HandleFunc("/orders/{id}/apply-ek", h.applyOrderEK).Methods(http.MethodPost)

	// Order tasks
	s.HandleFunc("/orders/{id}/tasks", h.listOrderTasks).Methods(http.MethodGet)
	s.HandleFunc("/orders/{id}/tasks", h.createOrderTask).Methods(http.MethodPost)
	s.HandleFunc("/orders/{id}/tasks/{taskId}", h.updateOrderTask).Methods(http.MethodPut)
	s.HandleFunc("/orders/{id}/tasks/{taskId}", h.deleteOrderTask).Methods(http.MethodDelete)

	// Order payments
	s.HandleFunc("/orders/{id}/payments", h.listOrderPayments).Methods(http.MethodGet)
	s.HandleFunc("/orders/{id}/payments", h.createOrderPayment).Methods(http.MethodPost)
	s.HandleFunc("/orders/{id}/payments/{paymentId}", h.updateOrderPayment).Methods(http.MethodPut)
	s.HandleFunc("/orders/{id}/payments/{paymentId}", h.deleteOrderPayment).Methods(http.MethodDelete)

	// Offers
	s.HandleFunc("/offers", h.listOffers).Methods(http.MethodGet)
	s.HandleFunc("/offers", h.createOffer).Methods(http.MethodPost)
	s.HandleFunc("/offers/{id}", h.getOffer).Methods(http.MethodGet)
	s.HandleFunc("/offers/{id}", h.updateOffer).Methods(http.MethodPut)
	s.HandleFunc("/offers/{id}", h.deleteOffer).Methods(http.MethodDelete)

	// Calendar entries (manual)
	s.HandleFunc("/calendar/entries", h.listCalendarEntries).Methods(http.MethodGet)
	s.HandleFunc("/calendar/entries", h.createCalendarEntry).Methods(http.MethodPost)
	s.HandleFunc("/calendar/entries/{id}", h.updateCalendarEntry).Methods(http.MethodPut)
	s.HandleFunc("/calendar/entries/{id}", h.deleteCalendarEntry).Methods(http.MethodDelete)

	// Task templates
	s.HandleFunc("/task-templates", h.listTaskTemplates).Methods(http.MethodGet)
	s.HandleFunc("/task-templates", h.createTaskTemplate).Methods(http.MethodPost)
	s.HandleFunc("/task-templates/{id}", h.updateTaskTemplate).Methods(http.MethodPut)
	s.HandleFunc("/task-templates/{id}", h.deleteTaskTemplate).Methods(http.MethodDelete)

	// Customers
	s.HandleFunc("/customers", h.listCustomers).Methods(http.MethodGet)
	s.HandleFunc("/customers", h.createCustomer).Methods(http.MethodPost)
	s.HandleFunc("/customers/{id}", h.getCustomer).Methods(http.MethodGet)
	s.HandleFunc("/customers/{id}", h.updateCustomer).Methods(http.MethodPut)
	s.HandleFunc("/customers/{id}", h.deleteCustomer).Methods(http.MethodDelete)

	// Warehouses (Lager)
	s.HandleFunc("/warehouses", h.listWarehouses).Methods(http.MethodGet)
	s.HandleFunc("/warehouses", h.createWarehouse).Methods(http.MethodPost)
	s.HandleFunc("/warehouses/{id}", h.getWarehouse).Methods(http.MethodGet)
	s.HandleFunc("/warehouses/{id}", h.updateWarehouse).Methods(http.MethodPut)
	s.HandleFunc("/warehouses/{id}", h.deleteWarehouse).Methods(http.MethodDelete)

	// Storage locations (Lagerplätze)
	s.HandleFunc("/storage-locations", h.listStorageLocations).Methods(http.MethodGet)
	s.HandleFunc("/storage-locations", h.createStorageLocation).Methods(http.MethodPost)
	s.HandleFunc("/storage-locations/{id}", h.getStorageLocation).Methods(http.MethodGet)
	s.HandleFunc("/storage-locations/{id}", h.updateStorageLocation).Methods(http.MethodPut)
	s.HandleFunc("/storage-locations/{id}", h.deleteStorageLocation).Methods(http.MethodDelete)

	// Stock movements (Wareneingang / Warenausgang)
	s.HandleFunc("/stock/movements", h.listStockMovements).Methods(http.MethodGet)
	s.HandleFunc("/stock/movements", h.createStockMovement).Methods(http.MethodPost)
	s.HandleFunc("/stock/movements/{id}", h.deleteStockMovement).Methods(http.MethodDelete)

	// Stock levels (Lagerbestand)
	s.HandleFunc("/stock/levels", h.listStockLevels).Methods(http.MethodGet)

	// Sales orders (Verkaufsaufträge – synced from Xentral; read-only in UI)
	s.HandleFunc("/sales-orders", h.listSalesOrders).Methods(http.MethodGet)
	s.HandleFunc("/sales-orders/{id}", h.getSalesOrder).Methods(http.MethodGet)

	// Supplier product configs (Lead Time, MOQ per supplier+product)
	s.HandleFunc("/supplier-product-configs", h.listSupplierProductConfigs).Methods(http.MethodGet)
	s.HandleFunc("/supplier-product-configs", h.upsertSupplierProductConfig).Methods(http.MethodPut)
	s.HandleFunc("/supplier-product-configs/{supplierId}/{productId}", h.deleteSupplierProductConfig).Methods(http.MethodDelete)

	// Xentral ERP integration
	NewXentralHandler(h.db, h.syslog).RegisterRoutes(s)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func procurementTenantID(r *http.Request) (primitive.ObjectID, bool) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok || tenant == nil {
		return primitive.NilObjectID, false
	}
	return tenant.ID, true
}

// parsePagination reads ?page=N&limit=N from the request.
// Returns page (1-based), limit, and skip.
// If limit is 0 the caller should load all documents.
func parsePagination(r *http.Request) (page, limit, skip int64) {
	limit = 25
	page = 1
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	if p, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64); err == nil && p > 0 {
		page = p
	}
	skip = (page - 1) * limit
	return
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func parseID(r *http.Request, key string) (primitive.ObjectID, bool) {
	s := mux.Vars(r)[key]
	id, err := primitive.ObjectIDFromHex(s)
	return id, err == nil
}

// ---------------------------------------------------------------------------
// Suppliers
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listSuppliers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"company": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"firstname": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"lastname": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"email": bson.M{"$regex": q, "$options": "i"}},
		}
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.Suppliers().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Suppliers().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "company", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Supplier, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Supplier
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.Suppliers().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Supplier
	err := h.db.Suppliers().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Supplier
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.UpdatedAt = time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"company":   doc.Company,
		"firstname": doc.Firstname,
		"lastname":  doc.Lastname,
		"email":     doc.Email,
		"skype":     doc.Skype,
		"origin":    doc.Origin,
		"misc":      doc.Misc,
		"tags":      doc.Tags,
		"updatedAt": doc.UpdatedAt,
	}}
	res, err := h.db.Suppliers().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}, update)
	if err != nil || res.MatchedCount == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.db.Suppliers().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID})
	if err != nil || res.DeletedCount == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// SupplierCodes
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listSupplierCodes(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if sid := r.URL.Query().Get("supplierId"); sid != "" {
		if oid, err := primitive.ObjectIDFromHex(sid); err == nil {
			filter["supplierId"] = oid
		}
	}
	cursor, err := h.db.SupplierCodes().Find(r.Context(), filter, options.Find().SetSort(bson.D{{Key: "short", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.SupplierCode, 0)
	if err := cursor.All(r.Context(), &results); err != nil {
		http.Error(w, "decode error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *ProcurementHandler) createSupplierCode(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.SupplierCode
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.SupplierCodes().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateSupplierCode(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.SupplierCode
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	res, err := h.db.SupplierCodes().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{"short": doc.Short, "supplierId": doc.SupplierID, "updatedAt": time.Now().UTC()}})
	if err != nil || res.MatchedCount == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteSupplierCode(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.SupplierCodes().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// GoodsGroups
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listGoodsGroups(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.GoodsGroups().CountDocuments(r.Context(), filter)
	cursor, err := h.db.GoodsGroups().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.GoodsGroup, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createGoodsGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.GoodsGroup
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.GoodsGroups().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateGoodsGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.GoodsGroup
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	h.db.GoodsGroups().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{"name": doc.Name, "short": doc.Short, "updatedAt": time.Now().UTC()}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteGoodsGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.GoodsGroups().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listProducts(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	andClauses := bson.A{}

	if q := r.URL.Query().Get("q"); q != "" {
		andClauses = append(andClauses, bson.M{"$or": bson.A{
			bson.M{"nameShort": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"ownNameShort": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"nameLong": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"ean": bson.M{"$regex": q, "$options": "i"}},
		}})
	}
	if sid := r.URL.Query().Get("supplierId"); sid != "" {
		if oid, err := primitive.ObjectIDFromHex(sid); err == nil {
			andClauses = append(andClauses, bson.M{"$or": bson.A{
				bson.M{"supplierId": oid},
				bson.M{"supplierIds": oid},
			}})
		}
	}
	if ggid := r.URL.Query().Get("goodsGroupId"); ggid != "" {
		if oid, err := primitive.ObjectIDFromHex(ggid); err == nil {
			filter["goodsGroupId"] = oid
		}
	}
	// Filter by tag (exact match, case-insensitive)
	if tag := r.URL.Query().Get("tag"); tag != "" {
		filter["tags"] = bson.M{"$regex": "^" + tag + "$", "$options": "i"}
	}
	// Filter by attribute key
	if attrKey := r.URL.Query().Get("attrKey"); attrKey != "" {
		attrFilter := bson.M{"attributes": bson.M{"$elemMatch": bson.M{"key": bson.M{"$regex": attrKey, "$options": "i"}}}}
		if attrVal := r.URL.Query().Get("attrValue"); attrVal != "" {
			attrFilter = bson.M{"attributes": bson.M{"$elemMatch": bson.M{
				"key":   bson.M{"$regex": attrKey, "$options": "i"},
				"value": bson.M{"$regex": attrVal, "$options": "i"},
			}}}
		}
		andClauses = append(andClauses, attrFilter)
	}

	if len(andClauses) > 0 {
		filter["$and"] = andClauses
	}
	page, pageSize, skip := parsePagination(r)
	total, _ := h.db.Products().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Products().Find(r.Context(), filter,
		options.Find().
			SetSort(bson.D{{Key: "nameShort", Value: 1}}).
			SetSkip(skip).
			SetLimit(pageSize))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	items := make([]models.Product, 0)
	cursor.All(r.Context(), &items) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"pages": (total + pageSize - 1) / pageSize,
	})
}

func (h *ProcurementHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Product
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.Products().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Product
	if err := h.db.Products().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Product
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = id
	doc.TenantID = tenantID
	doc.UpdatedAt = time.Now().UTC()
	h.db.Products().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": doc}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.Products().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// FreightCarriers
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listFreightCarriers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.FreightCarriers().CountDocuments(r.Context(), filter)
	cursor, err := h.db.FreightCarriers().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.FreightCarrier, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createFreightCarrier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.FreightCarrier
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.FreightCarriers().InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateFreightCarrier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.FreightCarrier
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	h.db.FreightCarriers().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{"name": doc.Name, "description": doc.Description, "updatedAt": time.Now().UTC()}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteFreightCarrier(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.FreightCarriers().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Harbours
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listHarbours(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.Harbours().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Harbours().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Harbour, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createHarbour(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Harbour
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.Harbours().InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateHarbour(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Harbour
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	h.db.Harbours().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{"name": doc.Name, "description": doc.Description, "updatedAt": time.Now().UTC()}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteHarbour(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.Harbours().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Containers
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listContainers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.ProcurementContainers().CountDocuments(r.Context(), filter)
	cursor, err := h.db.ProcurementContainers().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Container, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createContainer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Container
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.ProcurementContainers().InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateContainer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Container
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	h.db.ProcurementContainers().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{"name": doc.Name, "description": doc.Description, "volumeM3": doc.VolumeM3,
			"heightM": doc.HeightM, "lengthM": doc.LengthM, "widthM": doc.WidthM, "updatedAt": time.Now().UTC()}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteContainer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.ProcurementContainers().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Countries
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listCountries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.ProcurementCountries().CountDocuments(r.Context(), filter)
	cursor, err := h.db.ProcurementCountries().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Country, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createCountry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Country
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.ProcurementCountries().InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateCountry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Country
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	h.db.ProcurementCountries().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"name":           doc.Name,
			"iso2":           doc.ISO2,
			"iso3":           doc.ISO3,
			"isoNumeric":     doc.ISONumeric,
			"currency":       doc.Currency,
			"currencyCode":   doc.CurrencyCode,
			"currencySymbol": doc.CurrencySymbol,
			"phoneCode":      doc.PhoneCode,
			"region":         doc.Region,
			"capital":        doc.Capital,
			"updatedAt":      time.Now().UTC(),
		}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) seedCountries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// Only seed if no countries exist yet for this tenant
	existing, _ := h.db.ProcurementCountries().CountDocuments(r.Context(), bson.M{"tenantId": tenantID})
	if existing > 0 {
		writeJSON(w, http.StatusOK, map[string]any{"message": "already seeded", "existing": existing})
		return
	}
	now := time.Now().UTC()
	docs := make([]any, 0, len(allCountries))
	for _, c := range allCountries {
		docs = append(docs, models.Country{
			ID:             primitive.NewObjectID(),
			TenantID:       tenantID,
			Name:           c.Name,
			ISO2:           c.ISO2,
			ISO3:           c.ISO3,
			ISONumeric:     c.ISONumeric,
			Currency:       c.Currency,
			CurrencyCode:   c.CurrencyCode,
			CurrencySymbol: c.CurrencySymbol,
			PhoneCode:      c.PhoneCode,
			Region:         c.Region,
			Capital:        c.Capital,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	res, err := h.db.ProcurementCountries().InsertMany(r.Context(), docs)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"inserted": len(res.InsertedIDs)})
}

func (h *ProcurementHandler) deleteCountry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.ProcurementCountries().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"orderNumber": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"orderContents": bson.M{"$regex": q, "$options": "i"}},
		}
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.Orders().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Orders().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "orderDate", Value: -1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Order, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) nextOrderNumber(ctx context.Context, tenantID primitive.ObjectID) string {
	counterKey := "order_number_" + tenantID.Hex()
	var result struct {
		Value int64 `bson:"value"`
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	_ = h.db.Counters().FindOneAndUpdate(ctx,
		bson.M{"_id": counterKey},
		bson.M{"$inc": bson.M{"value": 1}},
		opts,
	).Decode(&result)
	year := time.Now().UTC().Year()
	return fmt.Sprintf("EK-%d-%04d", year, result.Value)
}

func (h *ProcurementHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Order
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	// Auto-generate internal number if not provided
	if doc.InternalNumber == "" {
		doc.InternalNumber = h.nextOrderNumber(r.Context(), tenantID)
	}
	// Set default statuses
	if doc.DeliveryStatus == "" {
		doc.DeliveryStatus = "pending"
	}
	if doc.PaymentStatus == "" {
		doc.PaymentStatus = "unpaid"
	}
	if doc.ReceiptStatus == "" {
		doc.ReceiptStatus = "pending"
	}
	if _, err := h.db.Orders().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.maybePushOrder(tenantID, doc.ID)
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Order
	if err := h.db.Orders().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// Migrate legacy single-freight field to freights array on first read
	if len(doc.Freights) == 0 && doc.Freight != nil {
		doc.Freights = []models.OrderFreight{*doc.Freight}
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Order
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = id
	doc.TenantID = tenantID
	doc.UpdatedAt = time.Now().UTC()
	h.db.Orders().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": doc}) //nolint
	h.maybePushOrder(tenantID, id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) patchOrderFreightDate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		Field string `json:"field"`
		Date  string `json:"date"`  // RFC3339 to set, empty string to clear
		Index int    `json:"index"` // index into freights array
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	allowed := map[string]bool{
		"shippingDate": true, "estimatedArrival": true, "arrival": true,
		"avisShipperDate": true, "docOfOrigin": true,
	}
	if !allowed[body.Field] {
		http.Error(w, "invalid field", http.StatusBadRequest)
		return
	}
	fieldPath := "freights." + strconv.Itoa(body.Index) + "." + body.Field
	if body.Date == "" {
		h.db.Orders().UpdateOne(r.Context(),
			bson.M{"_id": id, "tenantId": tenantID},
			bson.M{
				"$unset": bson.M{fieldPath: ""},
				"$set":   bson.M{"updatedAt": time.Now().UTC()},
			}) //nolint
	} else {
		t, err := time.Parse(time.RFC3339, body.Date)
		if err != nil {
			http.Error(w, "invalid date format", http.StatusBadRequest)
			return
		}
		h.db.Orders().UpdateOne(r.Context(),
			bson.M{"_id": id, "tenantId": tenantID},
			bson.M{"$set": bson.M{
				fieldPath:   t,
				"updatedAt": time.Now().UTC(),
			}}) //nolint
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.Orders().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// OrderTasks
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listOrderTasks(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orderID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}
	cursor, _ := h.db.OrderTasks().Find(r.Context(), bson.M{"tenantId": tenantID, "orderId": orderID},
		options.Find().SetSort(bson.D{{Key: "dueDate", Value: 1}}))
	results := make([]models.OrderTask, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, results)
}

func (h *ProcurementHandler) createOrderTask(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orderID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}
	var doc models.OrderTask
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.OrderID = orderID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.OrderTasks().InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateOrderTask(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "taskId")
	if !ok {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	var doc models.OrderTask
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.UpdatedAt = time.Now().UTC()
	h.db.OrderTasks().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"title":      doc.Title,
			"text":       doc.Text,
			"status":     doc.Status,
			"priority":   doc.Priority,
			"dueDate":    doc.DueDate,
			"doneAt":     doc.DoneAt,
			"doneById":   doc.DoneByID,
			"assigneeId": doc.AssigneeID,
			"updatedAt":  doc.UpdatedAt,
		}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteOrderTask(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "taskId")
	if !ok {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}
	h.db.OrderTasks().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// OrderPayments
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listOrderPayments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orderID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}
	cursor, _ := h.db.OrderPayments().Find(r.Context(), bson.M{"tenantId": tenantID, "orderId": orderID},
		options.Find().SetSort(bson.D{{Key: "nr", Value: 1}}))
	results := make([]models.OrderPayment, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, results)
}

func (h *ProcurementHandler) createOrderPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	orderID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}
	var doc models.OrderPayment
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.OrderID = orderID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.OrderPayments().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateOrderPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "paymentId")
	if !ok {
		http.Error(w, "invalid payment id", http.StatusBadRequest)
		return
	}
	var doc models.OrderPayment
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = id
	doc.TenantID = tenantID
	doc.UpdatedAt = time.Now().UTC()
	h.db.OrderPayments().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": doc}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteOrderPayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "paymentId")
	if !ok {
		http.Error(w, "invalid payment id", http.StatusBadRequest)
		return
	}
	h.db.OrderPayments().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Offers
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listOffers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if stockOnly := r.URL.Query().Get("stock"); stockOnly == "true" {
		filter["isStockOffer"] = true
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.Offers().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Offers().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Offer, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createOffer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Offer
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.Offers().InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getOffer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Offer
	if err := h.db.Offers().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateOffer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Offer
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = id
	doc.TenantID = tenantID
	doc.UpdatedAt = time.Now().UTC()
	h.db.Offers().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": doc}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteOffer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.Offers().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// EK-Kalkulation (Warenbezugskosten)
// ---------------------------------------------------------------------------

// freightCostEUR returns the total landed cost in EUR for a single OrderFreight,
// using freightDollarRate as fallback when f.DollarRate == 0.
func freightCostEUR(f models.OrderFreight, fallbackRate float64) float64 {
	rate := f.DollarRate
	if rate <= 0 {
		rate = fallbackRate
	}
	frtEUR := f.FreightageEUR
	if frtEUR == 0 {
		frtEUR = f.PreFreightageEUR
	}
	seaUSD := f.SeaFreightUSD + f.EmergencyBunkerSurchargeUSD +
		f.PeakSeasonSurchargeUSD + f.SuezCanalAddonUSD + f.DangerPayUSD
	return frtEUR +
		seaUSD*rate +
		f.THCEUR + f.ISPSEUR + f.BLDocFeeEUR + f.FollowUpFeesEUR +
		f.CustomsClearanceEUR + f.CustomsEUR +
		f.ContainerBookingCostEUR
}

// applyOrderEK calculates the landed cost (EK) per product per container.
//
// When an OrderProduct has FreightIndex set, its freight share comes only from
// that container's costs and is volume-proportional within that container.
// When FreightIndex is nil (legacy / unassigned), costs are spread across all
// containers proportional to volume — preserving the original behaviour.
//
// For every processed product the function:
//  1. Updates products.lastEk / lastEkDate
//  2. Upserts a ProductPriceList EK record (name = order number + container)
//  3. Upserts an InventoryLot so FIFO/LIFO valuation can walk the batches
func (h *ProcurementHandler) applyOrderEK(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := h.db.Orders().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&order); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// --- Dollar rate: average of all payment rates; fallback to preDollarRate ---
	var payments []models.OrderPayment
	cur, err := h.db.OrderPayments().Find(r.Context(), bson.M{"orderId": id, "tenantId": tenantID})
	if err == nil {
		_ = cur.All(r.Context(), &payments)
	}

	dollarRateAvg := order.PreDollarRate
	totalPaymentFees := 0.0
	if len(payments) > 0 {
		rateSum := 0.0
		for _, p := range payments {
			rateSum += p.PaymentDollarRate
			totalPaymentFees += p.PaymentFees
		}
		if avg := rateSum / float64(len(payments)); avg > 0 {
			dollarRateAvg = avg
		}
	}
	if dollarRateAvg <= 0 {
		http.Error(w, "Dollarkurs nicht ermittelbar — Zahlungen oder preDollarRate angeben", http.StatusBadRequest)
		return
	}

	// Migrate legacy single-freight if needed
	if len(order.Freights) == 0 && order.Freight != nil {
		order.Freights = []models.OrderFreight{*order.Freight}
	}

	// --- Pre-compute per-container freight EUR and per-container volume totals ---
	containerFreightEUR := make([]float64, len(order.Freights))
	for i, f := range order.Freights {
		containerFreightEUR[i] = freightCostEUR(f, dollarRateAvg)
	}

	// Volume sums: one entry per container (index) + one catch-all for unassigned
	// containerVol[i] = total m³ of products assigned to freights[i]
	// unassignedVol   = total m³ of products without a FreightIndex
	containerVol := make([]float64, len(order.Freights))
	containerQty := make([]int, len(order.Freights))
	unassignedVol := 0.0
	unassignedQty := 0
	for _, op := range order.Products {
		if op.FreightIndex != nil && *op.FreightIndex >= 0 && *op.FreightIndex < len(order.Freights) {
			containerVol[*op.FreightIndex] += op.VolumeM3 * float64(op.Quantity)
			containerQty[*op.FreightIndex] += op.Quantity
		} else {
			unassignedVol += op.VolumeM3 * float64(op.Quantity)
			unassignedQty += op.Quantity
		}
	}

	// Total freight for unassigned products = sum of all containers
	totalFreightEUR := 0.0
	for _, c := range containerFreightEUR {
		totalFreightEUR += c
	}

	// Discount and orderSumUSD for price factor calculations
	now := time.Now().UTC()
	orderRef := order.ID

	updated := 0

	for _, op := range order.Products {
		if op.ProductID.IsZero() || op.Quantity <= 0 {
			continue
		}

		// --- Determine which freight costs and volume pool to use ---
		var relevantFreightEUR float64
		var volumePool float64
		var qtyPool int
		var freightIdx *int

		if op.FreightIndex != nil && *op.FreightIndex >= 0 && *op.FreightIndex < len(order.Freights) {
			idx := *op.FreightIndex
			relevantFreightEUR = containerFreightEUR[idx]
			volumePool = containerVol[idx]
			qtyPool = containerQty[idx]
			freightIdx = op.FreightIndex
		} else {
			// Legacy: spread across all containers
			relevantFreightEUR = totalFreightEUR
			volumePool = unassignedVol
			qtyPool = unassignedQty
		}

		// Volume factor within the relevant pool
		prodVol := op.VolumeM3 * float64(op.Quantity)
		var volumeFactor float64
		if volumePool > 0 {
			volumeFactor = prodVol / volumePool
		} else if qtyPool > 0 {
			volumeFactor = float64(op.Quantity) / float64(qtyPool)
		}

		// Price factor for payment fees
		priceFactor := 0.0
		if order.OrderSumUSD > 0 {
			priceFactor = op.UnitPriceUSD / order.OrderSumUSD
		}

		freightShare := relevantFreightEUR * volumeFactor
		feesShare := totalPaymentFees * priceFactor

		// WBK (Waren-Bezugskosten) with transport insurance
		wbk := (freightShare + feesShare) * (1000 + order.TransportInsurancePermille) / 1000

		// Discount proportional to product's price share
		discount := 0.0
		if order.OrderSumUSD+order.Discount > 0 {
			discount = order.Discount / (order.OrderSumUSD + order.Discount) * op.UnitPriceUSD
		}

		unitEkEUR := math.Round((wbk/float64(op.Quantity)+(op.UnitPriceUSD-discount)/dollarRateAvg)*10000) / 10000

		// 1. Update product.lastEk
		if _, err := h.db.Products().UpdateOne(
			r.Context(),
			bson.M{"_id": op.ProductID, "tenantId": tenantID},
			bson.M{"$set": bson.M{"lastEk": unitEkEUR, "lastEkDate": now, "updatedAt": now}},
		); err == nil {
			updated++
		}

		// 2. Upsert ProductPriceList EK record — one per product per container (or "all")
		containerLabel := "Alle Container"
		if freightIdx != nil {
			containerLabel = fmt.Sprintf("Container %d", *freightIdx+1)
			if *freightIdx < len(order.Freights) && order.Freights[*freightIdx].ContainerNr != "" {
				containerLabel = order.Freights[*freightIdx].ContainerNr
			}
		}
		plName := fmt.Sprintf("EK %s / %s", order.OrderNumber, containerLabel)
		plFilter := bson.M{
			"tenantId":  tenantID,
			"productId": op.ProductID,
			"type":      "EK",
			"orderId":   orderRef,
		}
		if freightIdx != nil {
			plFilter["freightIndex"] = *freightIdx
		} else {
			plFilter["freightIndex"] = bson.M{"$exists": false}
		}
		plUpdate := bson.M{"$set": bson.M{
			"name":         plName,
			"price":        unitEkEUR,
			"currency":     "EUR",
			"quantity":     op.Quantity,
			"source":       "order",
			"orderId":      orderRef,
			"freightIndex": freightIdx,
			"validFrom":    now,
			"updatedAt":    now,
		}, "$setOnInsert": bson.M{
			"tenantId":  tenantID,
			"productId": op.ProductID,
			"type":      "EK",
			"createdAt": now,
		}}
		upsertTrue := true
		if _, err := h.db.ProductPriceLists().UpdateOne(r.Context(), plFilter, plUpdate,
			&options.UpdateOptions{Upsert: &upsertTrue}); err != nil {
			_ = err // price list upsert is best-effort
		}

		// 3. Upsert InventoryLot — one per product per container
		lotFilter := bson.M{
			"tenantId":  tenantID,
			"productId": op.ProductID,
			"orderId":   orderRef,
			"source":    "order",
		}
		if freightIdx != nil {
			lotFilter["freightIndex"] = *freightIdx
		} else {
			lotFilter["freightIndex"] = bson.M{"$exists": false}
		}
		// Determine receivedAt: use container arrival date if available, else now
		receivedAt := now
		if freightIdx != nil && *freightIdx < len(order.Freights) {
			if arr := order.Freights[*freightIdx].Arrival; arr != nil {
				receivedAt = *arr
			} else if eta := order.Freights[*freightIdx].EstimatedArrival; eta != nil {
				receivedAt = *eta
			}
		}
		lotUpdate := bson.M{"$set": bson.M{
			"quantity":   op.Quantity,
			"unitEkEur":  unitEkEUR,
			"receivedAt": receivedAt,
			"updatedAt":  now,
		}, "$setOnInsert": bson.M{
			"tenantId":  tenantID,
			"productId": op.ProductID,
			"orderId":   orderRef,
			"source":    "order",
			"remaining": op.Quantity,
			"createdAt": now,
		}}
		if freightIdx != nil {
			lotUpdate["$set"].(bson.M)["freightIndex"] = *freightIdx
			lotUpdate["$setOnInsert"].(bson.M)["freightIndex"] = *freightIdx
		}
		if _, err := h.db.InventoryLots().UpdateOne(r.Context(), lotFilter, lotUpdate,
			&options.UpdateOptions{Upsert: &upsertTrue}); err != nil {
			_ = err // lot upsert is best-effort
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"updated": updated})

	// Auto-push EK prices to Xentral if the tenant has it configured.
	// Runs async so the HTTP response is not delayed.
	go func(tID, oID primitive.ObjectID) {
		pushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		var xCfg models.XentralConfig
		if err := h.db.XentralConfigs().FindOne(pushCtx, bson.M{
			"tenantId": tID, "enabled": true, "pushProductsToXentral": true,
		}).Decode(&xCfg); err != nil {
			return // not configured or disabled
		}
		cur, err := h.db.ProductPriceLists().Find(pushCtx, bson.M{
			"tenantId": tID,
			"orderId":  oID,
			"source":   "order",
			"type":     "EK",
		})
		if err != nil {
			return
		}
		var prices []models.ProductPriceList
		_ = cur.All(pushCtx, &prices)
		client := xentral.NewClient(xCfg.BaseURL, xCfg.APIToken)
		for _, p := range prices {
			if _, err := h.xentral.PushPurchasePriceToXentral(pushCtx, tID, p.ID, client); err != nil {
				h.syslog.Log(pushCtx, "medium", fmt.Sprintf(
					"auto-push EK to Xentral failed: order=%s product=%s err=%v",
					oID.Hex(), p.ProductID.Hex(), err,
				))
			}
		}
	}(tenantID, orderRef)
}

// ---------------------------------------------------------------------------
// Product Price Lists
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listProductPriceLists(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	filter := bson.M{"tenantId": tenantID, "productId": productID}
	cursor, err := h.db.ProductPriceLists().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "type", Value: 1}, {Key: "name", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.ProductPriceList, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, results)
}

func (h *ProcurementHandler) createProductPriceList(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.ProductPriceList
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.ProductID = productID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.ProductPriceLists().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateProductPriceList(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	priceListID, ok := parseID(r, "priceListId")
	if !ok {
		http.Error(w, "invalid priceListId", http.StatusBadRequest)
		return
	}
	var doc models.ProductPriceList
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	h.db.ProductPriceLists().UpdateOne(r.Context(), bson.M{"_id": priceListID, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"type":      doc.Type,
			"name":      doc.Name,
			"price":     doc.Price,
			"currency":  doc.Currency,
			"validFrom": doc.ValidFrom,
			"validTo":   doc.ValidTo,
			"notes":     doc.Notes,
			"updatedAt": time.Now().UTC(),
		}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteProductPriceList(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	priceListID, ok := parseID(r, "priceListId")
	if !ok {
		http.Error(w, "invalid priceListId", http.StatusBadRequest)
		return
	}
	h.db.ProductPriceLists().DeleteOne(r.Context(), bson.M{"_id": priceListID, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// EK History (ProductPriceList, filtered to type=EK, with chart aggregation)
// ---------------------------------------------------------------------------

// getEKHistory returns all EK price-list entries for a product in chronological
// order, optionally filtered by date range.
//
// GET /products/:id/ek-history?from=YYYY-MM-DD&to=YYYY-MM-DD&format=json|csv
func (h *ProcurementHandler) getEKHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	filter := bson.M{"tenantId": tenantID, "productId": productID, "type": "EK"}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			if filter["validFrom"] == nil {
				filter["validFrom"] = bson.M{}
			}
			filter["validFrom"].(bson.M)["$gte"] = t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			t = t.Add(24 * time.Hour)
			if filter["validFrom"] == nil {
				filter["validFrom"] = bson.M{}
			}
			filter["validFrom"].(bson.M)["$lte"] = t
		}
	}

	cursor, err := h.db.ProductPriceLists().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "validFrom", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var rows []models.ProductPriceList
	cursor.All(r.Context(), &rows) //nolint

	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"ek-history-%s.csv\"", productID.Hex()))
		fmt.Fprintf(w, "datum,ek_eur,menge,container,bestellnummer,quelle\n")
		for _, row := range rows {
			date := ""
			if row.ValidFrom != nil {
				date = row.ValidFrom.Format("2006-01-02")
			}
			container := ""
			if row.FreightIndex != nil {
				container = fmt.Sprintf("%d", *row.FreightIndex+1)
			}
			orderNr := ""
			if row.OrderID != nil {
				orderNr = row.OrderID.Hex()
			}
			fmt.Fprintf(w, "%s,%.4f,%d,%s,%s,%s\n", date, row.Price, row.Quantity, container, orderNr, row.Source)
		}
		return
	}

	writeJSON(w, http.StatusOK, rows)
}

// getEKHistoryChart returns time-series data suitable for rendering a line chart.
//
// GET /products/:id/ek-history/chart
func (h *ProcurementHandler) getEKHistoryChart(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	cursor, err := h.db.ProductPriceLists().Find(r.Context(),
		bson.M{"tenantId": tenantID, "productId": productID, "type": "EK"},
		options.Find().SetSort(bson.D{{Key: "validFrom", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var rows []models.ProductPriceList
	cursor.All(r.Context(), &rows) //nolint

	type chartPoint struct {
		Date         string  `json:"date"`
		Price        float64 `json:"price"`
		Quantity     int     `json:"quantity"`
		FreightIndex *int    `json:"freightIndex,omitempty"`
		Source       string  `json:"source"`
	}
	points := make([]chartPoint, 0, len(rows))
	for _, row := range rows {
		date := ""
		if row.ValidFrom != nil {
			date = row.ValidFrom.Format("2006-01-02")
		}
		points = append(points, chartPoint{
			Date:         date,
			Price:        row.Price,
			Quantity:     row.Quantity,
			FreightIndex: row.FreightIndex,
			Source:       row.Source,
		})
	}
	writeJSON(w, http.StatusOK, points)
}

// ---------------------------------------------------------------------------
// Inventory Lots — CRUD + import
// ---------------------------------------------------------------------------

// listInventoryLots returns all lots for a product, sorted oldest-first (FIFO order).
//
// GET /products/:id/inventory-lots
func (h *ProcurementHandler) listInventoryLots(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	cursor, err := h.db.InventoryLots().Find(r.Context(),
		bson.M{"tenantId": tenantID, "productId": productID},
		options.Find().SetSort(bson.D{{Key: "receivedAt", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var lots []models.InventoryLot
	cursor.All(r.Context(), &lots) //nolint
	writeJSON(w, http.StatusOK, lots)
}

// importInventoryLots handles CSV bulk import for initial stock take-over.
//
// POST /inventory/import
// Body: CSV with columns: productId,menge,ek_eur,datum,notizen
func (h *ProcurementHandler) importInventoryLots(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "form parse error", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	type importRow struct {
		ProductIDHex string
		Quantity     int
		UnitEkEUR    float64
		ReceivedAt   time.Time
		Notes        string
	}

	now := time.Now().UTC()
	imported := 0
	errors := []string{}

	// Read CSV line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++
		if lineNum == 1 {
			continue // skip header
		}
		if line == "" {
			continue
		}
		parts := splitCSV(line)
		if len(parts) < 3 {
			errors = append(errors, fmt.Sprintf("Zeile %d: zu wenige Felder", lineNum))
			continue
		}
		productOID, pErr := primitive.ObjectIDFromHex(strings.TrimSpace(parts[0]))
		if pErr != nil {
			errors = append(errors, fmt.Sprintf("Zeile %d: ungültige productId", lineNum))
			continue
		}
		qty := 0
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &qty)
		if qty <= 0 {
			errors = append(errors, fmt.Sprintf("Zeile %d: Menge muss > 0 sein", lineNum))
			continue
		}
		ek := 0.0
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%f", &ek)

		receivedAt := now
		if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
			if t, tErr := time.Parse("2006-01-02", strings.TrimSpace(parts[3])); tErr == nil {
				receivedAt = t
			}
		}
		notes := ""
		if len(parts) >= 5 {
			notes = strings.TrimSpace(parts[4])
		}

		lot := models.InventoryLot{
			TenantID:   tenantID,
			ProductID:  productOID,
			ReceivedAt: receivedAt,
			Quantity:   qty,
			Remaining:  qty,
			UnitEkEUR:  ek,
			Source:     "import",
			Notes:      notes,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if _, iErr := h.db.InventoryLots().InsertOne(r.Context(), lot); iErr != nil {
			errors = append(errors, fmt.Sprintf("Zeile %d: DB-Fehler %v", lineNum, iErr))
			continue
		}
		// Also update product.lastEk with import value
		h.db.Products().UpdateOne(r.Context(), //nolint
			bson.M{"_id": productOID, "tenantId": tenantID},
			bson.M{"$set": bson.M{"lastEk": ek, "lastEkDate": receivedAt, "updatedAt": now}})
		imported++
	}

	writeJSON(w, http.StatusOK, map[string]any{"imported": imported, "errors": errors})
}

// exportInventoryValuation exports the current stock valuation as CSV.
//
// GET /inventory/export?method=fifo|lifo|weighted_avg
func (h *ProcurementHandler) exportInventoryValuation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	method := r.URL.Query().Get("method")
	if method == "" {
		method = "fifo"
	}

	// Fetch all lots with remaining > 0
	cursor, err := h.db.InventoryLots().Find(r.Context(),
		bson.M{"tenantId": tenantID, "remaining": bson.M{"$gt": 0}},
		options.Find().SetSort(bson.D{{Key: "productId", Value: 1}, {Key: "receivedAt", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var lots []models.InventoryLot
	cursor.All(r.Context(), &lots) //nolint

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"lagerbewertung-%s-%s.csv\"",
		method, time.Now().Format("2006-01-02")))
	fmt.Fprintf(w, "produktId,menge,ek_eur,gesamt_eur,wareneingangsdatum,quelle,methode\n")
	for _, lot := range lots {
		fmt.Fprintf(w, "%s,%d,%.4f,%.4f,%s,%s,%s\n",
			lot.ProductID.Hex(),
			lot.Remaining,
			lot.UnitEkEUR,
			float64(lot.Remaining)*lot.UnitEkEUR,
			lot.ReceivedAt.Format("2006-01-02"),
			lot.Source,
			method,
		)
	}
}

// splitCSV splits a CSV line respecting simple quoting (no embedded newlines).
func splitCSV(line string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for _, ch := range line {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ',' && !inQuote:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(ch)
		}
	}
	parts = append(parts, cur.String())
	return parts
}

// ---------------------------------------------------------------------------
// Calendar
// ---------------------------------------------------------------------------

type calendarTask struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"orderId"`
	OrderNumber string  `json:"orderNumber"`
	Text        string  `json:"text"`
	DueDate     string  `json:"dueDate"`
	DoneAt      *string `json:"doneAt"`
}

type calendarArrival struct {
	OrderID       string `json:"orderId"`
	OrderNumber   string `json:"orderNumber"`
	FreightIndex  int    `json:"freightIndex"`
	EstimatedDate string `json:"estimatedDate"`
	ActualDate    string `json:"actualDate,omitempty"`
	SupplierName  string `json:"supplierName,omitempty"`
}

func (h *ProcurementHandler) getCalendar(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	var from, to time.Time
	var err error
	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			http.Error(w, "invalid from date", http.StatusBadRequest)
			return
		}
	} else {
		now := time.Now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			http.Error(w, "invalid to date", http.StatusBadRequest)
			return
		}
		to = to.Add(24 * time.Hour) // inclusive
	} else {
		to = from.AddDate(0, 1, 0)
	}

	// Fetch tasks with dueDate in range
	taskCursor, err := h.db.OrderTasks().Find(r.Context(), bson.M{
		"tenantId": tenantID,
		"dueDate":  bson.M{"$gte": from, "$lt": to},
	})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	rawTasks := make([]models.OrderTask, 0)
	taskCursor.All(r.Context(), &rawTasks) //nolint

	// Collect orderIds for enrichment
	orderIDSet := map[primitive.ObjectID]struct{}{}
	for _, t := range rawTasks {
		orderIDSet[t.OrderID] = struct{}{}
	}

	// Fetch orders with estimatedArrival or arrival in range (freights array or legacy freight)
	arrivalCursor, err := h.db.Orders().Find(r.Context(), bson.M{
		"tenantId": tenantID,
		"$or": bson.A{
			bson.M{"freights.estimatedArrival": bson.M{"$gte": from, "$lt": to}},
			bson.M{"freights.arrival": bson.M{"$gte": from, "$lt": to}},
			bson.M{"freight.estimatedArrival": bson.M{"$gte": from, "$lt": to}},
			bson.M{"freight.arrival": bson.M{"$gte": from, "$lt": to}},
		},
	})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	arrivalOrders := make([]models.Order, 0)
	arrivalCursor.All(r.Context(), &arrivalOrders) //nolint
	for _, o := range arrivalOrders {
		orderIDSet[o.ID] = struct{}{}
	}

	// Fetch all needed orders in one query
	orderIDs := make([]primitive.ObjectID, 0, len(orderIDSet))
	for id := range orderIDSet {
		orderIDs = append(orderIDs, id)
	}
	orderMap := map[primitive.ObjectID]models.Order{}
	if len(orderIDs) > 0 {
		oCursor, err2 := h.db.Orders().Find(r.Context(), bson.M{"_id": bson.M{"$in": orderIDs}})
		if err2 == nil {
			var orders []models.Order
			oCursor.All(r.Context(), &orders) //nolint
			for _, o := range orders {
				orderMap[o.ID] = o
			}
		}
	}

	// Optionally fetch supplier names
	supplierIDs := make([]primitive.ObjectID, 0)
	for _, o := range orderMap {
		if o.SupplierID != nil {
			supplierIDs = append(supplierIDs, *o.SupplierID)
		}
	}
	supplierMap := map[primitive.ObjectID]string{}
	if len(supplierIDs) > 0 {
		sCursor, err2 := h.db.Suppliers().Find(r.Context(), bson.M{"_id": bson.M{"$in": supplierIDs}})
		if err2 == nil {
			var suppliers []models.Supplier
			sCursor.All(r.Context(), &suppliers) //nolint
			for _, s := range suppliers {
				supplierMap[s.ID] = s.Company
			}
		}
	}

	// Build response tasks
	tasks := make([]calendarTask, 0, len(rawTasks))
	for _, t := range rawTasks {
		ct := calendarTask{
			ID:      t.ID.Hex(),
			OrderID: t.OrderID.Hex(),
			Text:    t.Text,
		}
		if t.DueDate != nil {
			ct.DueDate = t.DueDate.Format("2006-01-02")
		}
		if t.DoneAt != nil {
			s := t.DoneAt.Format(time.RFC3339)
			ct.DoneAt = &s
		}
		if o, ok2 := orderMap[t.OrderID]; ok2 {
			ct.OrderNumber = o.OrderNumber
		}
		tasks = append(tasks, ct)
	}

	// Build response arrivals — one entry per container per order
	arrivals := make([]calendarArrival, 0, len(arrivalOrders))
	for _, o := range arrivalOrders {
		// Migrate legacy single freight
		freights := o.Freights
		if len(freights) == 0 && o.Freight != nil {
			freights = []models.OrderFreight{*o.Freight}
		}
		supplierName := ""
		if o.SupplierID != nil {
			supplierName = supplierMap[*o.SupplierID]
		}
		for fi, f := range freights {
			if f.EstimatedArrival == nil && f.Arrival == nil {
				continue
			}
			ca := calendarArrival{
				OrderID:      o.ID.Hex(),
				OrderNumber:  o.OrderNumber,
				FreightIndex: fi,
				SupplierName: supplierName,
			}
			if f.EstimatedArrival != nil {
				ca.EstimatedDate = f.EstimatedArrival.Format("2006-01-02")
			}
			if f.Arrival != nil {
				ca.ActualDate = f.Arrival.Format("2006-01-02")
			}
			arrivals = append(arrivals, ca)
		}
	}

	// Fetch manual calendar entries for range
	entryCursor, err2 := h.db.CalendarEntries().Find(r.Context(), bson.M{
		"tenantId": tenantID,
		"date":     bson.M{"$gte": from, "$lt": to},
	}, options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	entries := make([]models.CalendarEntry, 0)
	if err2 == nil {
		entryCursor.All(r.Context(), &entries) //nolint
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tasks":    tasks,
		"arrivals": arrivals,
		"entries":  entries,
	})
}

// ---------------------------------------------------------------------------
// Calendar entries (manual)
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listCalendarEntries(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	filter := bson.M{"tenantId": tenantID}
	if fromStr != "" || toStr != "" {
		dateFilter := bson.M{}
		if fromStr != "" {
			if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				dateFilter["$gte"] = t
			}
		}
		if toStr != "" {
			if t, err := time.Parse("2006-01-02", toStr); err == nil {
				dateFilter["$lt"] = t.Add(24 * time.Hour)
			}
		}
		if len(dateFilter) > 0 {
			filter["date"] = dateFilter
		}
	}
	cursor, err := h.db.CalendarEntries().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.CalendarEntry, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, results)
}

func (h *ProcurementHandler) createCalendarEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.CalendarEntry
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.CalendarEntries().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateCalendarEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.CalendarEntry
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	h.db.CalendarEntries().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"title":     doc.Title,
			"notes":     doc.Notes,
			"date":      doc.Date,
			"entryType": doc.EntryType,
			"updatedAt": time.Now().UTC(),
		}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteCalendarEntry(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.CalendarEntries().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Task Templates
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listTaskTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	cursor, _ := h.db.OrderTasks().Database().Collection("task_templates").Find(r.Context(),
		bson.M{"tenantId": tenantID},
		options.Find().SetSort(bson.D{{Key: "phase", Value: 1}, {Key: "daysAfter", Value: 1}}))
	results := make([]models.TaskTemplate, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, results)
}

func (h *ProcurementHandler) createTaskTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.TaskTemplate
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	h.db.OrderTasks().Database().Collection("task_templates").InsertOne(r.Context(), doc) //nolint
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) updateTaskTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.TaskTemplate
	json.NewDecoder(r.Body).Decode(&doc) //nolint
	doc.UpdatedAt = time.Now().UTC()
	h.db.OrderTasks().Database().Collection("task_templates").UpdateOne(r.Context(),
		bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{"text": doc.Text, "daysAfter": doc.DaysAfter, "phase": doc.Phase, "updatedAt": doc.UpdatedAt}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteTaskTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.OrderTasks().Database().Collection("task_templates").DeleteOne(r.Context(),
		bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Customers
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listCustomers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"company": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"firstname": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"lastname": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"email": bson.M{"$regex": q, "$options": "i"}},
		}
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.Customers().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Customers().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "company", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.Customer, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Customer
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.Customers().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Customer
	if err := h.db.Customers().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Customer
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.UpdatedAt = time.Now().UTC()
	h.db.Customers().UpdateOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"company":   doc.Company,
			"firstname": doc.Firstname,
			"lastname":  doc.Lastname,
			"email":     doc.Email,
			"phone":     doc.Phone,
			"address":   doc.Address,
			"misc":      doc.Misc,
			"tags":      doc.Tags,
			"updatedAt": doc.UpdatedAt,
		}}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteCustomer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	h.db.Customers().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Stock Movements & Levels
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listStockMovements(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if pid := r.URL.Query().Get("productId"); pid != "" {
		if oid, err := primitive.ObjectIDFromHex(pid); err == nil {
			filter["productId"] = oid
		}
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter["type"] = t
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.StockMovements().CountDocuments(r.Context(), filter)
	cursor, err := h.db.StockMovements().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "movedAt", Value: -1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.StockMovement, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createStockMovement(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var userID primitive.ObjectID
	if u, ok := middleware.GetUserFromContext(r.Context()); ok {
		userID = u.ID
	}
	var doc models.StockMovement
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if doc.ProductID.IsZero() {
		http.Error(w, "productId required", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.ProcessedBy = userID
	if doc.MovedAt.IsZero() {
		doc.MovedAt = time.Now().UTC()
	}
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt

	if _, err := h.db.StockMovements().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// Update stock level
	delta := doc.Quantity
	if doc.Type == "issue" {
		delta = -doc.Quantity
	}
	levelOpts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var level models.StockLevel
	_ = h.db.StockLevels().FindOneAndUpdate(r.Context(),
		bson.M{"tenantId": tenantID, "productId": doc.ProductID},
		bson.M{
			"$inc": bson.M{"quantity": delta},
			"$set": bson.M{"updatedAt": time.Now().UTC(), "unit": doc.Unit, "location": doc.Location},
			"$setOnInsert": bson.M{"tenantId": tenantID, "productId": doc.ProductID},
		},
		levelOpts,
	).Decode(&level)

	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) deleteStockMovement(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	// Fetch movement first to reverse the stock delta
	var doc models.StockMovement
	if err := h.db.StockMovements().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.db.StockMovements().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}) //nolint

	// Reverse stock level update
	delta := -doc.Quantity
	if doc.Type == "issue" {
		delta = doc.Quantity
	}
	h.db.StockLevels().UpdateOne(r.Context(),
		bson.M{"tenantId": tenantID, "productId": doc.ProductID},
		bson.M{"$inc": bson.M{"quantity": delta}, "$set": bson.M{"updatedAt": time.Now().UTC()}}) //nolint

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) listStockLevels(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if pid := r.URL.Query().Get("productId"); pid != "" {
		if oid, err := primitive.ObjectIDFromHex(pid); err == nil {
			filter["productId"] = oid
		}
	}
	// Optionally only show non-zero levels
	if r.URL.Query().Get("nonZero") == "true" {
		filter["quantity"] = bson.M{"$ne": 0}
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.StockLevels().CountDocuments(r.Context(), filter)
	cursor, err := h.db.StockLevels().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.StockLevel, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

// ---------------------------------------------------------------------------
// Inventory valuation
// ---------------------------------------------------------------------------

type valuationResult struct {
	UnitEkEUR    float64 `json:"unitEkEur"`
	TotalValueEur float64 `json:"totalValueEur"`
}

type productInventoryValuation struct {
	TotalQuantity int              `json:"totalQuantity"`
	LastEK        float64          `json:"lastEk"`
	LastEKDate    *time.Time       `json:"lastEkDate,omitempty"`
	FIFO          valuationResult  `json:"fifo"`
	LIFO          valuationResult  `json:"lifo"`
	WeightedAvg   valuationResult  `json:"weightedAvg"`
	EkDbMethode   string           `json:"ekDbMethode"`
}

// getInventoryValuation returns the current stock valuation for a product
// using three methods (FIFO, LIFO, weighted average) plus the last EK.
//
// GET /products/:id/inventory-valuation
func (h *ProcurementHandler) getInventoryValuation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	productID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Load product for LastEK
	var product models.Product
	if err := h.db.Products().FindOne(r.Context(), bson.M{"_id": productID, "tenantId": tenantID}).Decode(&product); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	// Load all inventory lots sorted oldest-first (FIFO order)
	cursor, err := h.db.InventoryLots().Find(r.Context(),
		bson.M{"tenantId": tenantID, "productId": productID},
		options.Find().SetSort(bson.D{{Key: "receivedAt", Value: 1}}))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var lots []models.InventoryLot
	cursor.All(r.Context(), &lots) //nolint

	// Totals
	totalRemaining := 0
	totalOriginalQty := 0
	totalOriginalValue := 0.0
	for _, l := range lots {
		totalRemaining += l.Remaining
		totalOriginalQty += l.Quantity
		totalOriginalValue += float64(l.Quantity) * l.UnitEkEUR
	}

	// FIFO: lot.Remaining already reflects FIFO consumption (oldest consumed first)
	fifoValue := 0.0
	for _, l := range lots {
		fifoValue += float64(l.Remaining) * l.UnitEkEUR
	}
	fifoUnitEk := 0.0
	if totalRemaining > 0 {
		fifoUnitEk = math.Round(fifoValue/float64(totalRemaining)*10000) / 10000
	}

	// LIFO: oldest lots remain in stock (newest consumed first).
	// Walk lots oldest→newest, fill up totalRemaining from the front.
	lifoValue := 0.0
	lifoToFill := totalRemaining
	for _, l := range lots {
		if lifoToFill <= 0 {
			break
		}
		take := l.Quantity
		if take > lifoToFill {
			take = lifoToFill
		}
		lifoValue += float64(take) * l.UnitEkEUR
		lifoToFill -= take
	}
	lifoUnitEk := 0.0
	if totalRemaining > 0 {
		lifoUnitEk = math.Round(lifoValue/float64(totalRemaining)*10000) / 10000
	}

	// Weighted average: average unit EK across all lots ever received,
	// independent of what has been consumed.
	wavgUnitEk := 0.0
	if totalOriginalQty > 0 {
		wavgUnitEk = math.Round(totalOriginalValue/float64(totalOriginalQty)*10000) / 10000
	}
	wavgValue := wavgUnitEk * float64(totalRemaining)

	// Load tenant config to include active method in response
	var cfg models.ProcurementTenantConfig
	if err := h.db.ProcurementTenantConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
		cfg.EkDbMethode = "fifo" // default
	}

	writeJSON(w, http.StatusOK, productInventoryValuation{
		TotalQuantity: totalRemaining,
		LastEK:        product.LastEK,
		LastEKDate:    product.LastEKDate,
		FIFO:          valuationResult{UnitEkEUR: fifoUnitEk, TotalValueEur: math.Round(fifoValue*100) / 100},
		LIFO:          valuationResult{UnitEkEUR: lifoUnitEk, TotalValueEur: math.Round(lifoValue*100) / 100},
		WeightedAvg:   valuationResult{UnitEkEUR: wavgUnitEk, TotalValueEur: math.Round(wavgValue*100) / 100},
		EkDbMethode:   cfg.EkDbMethode,
	})
}

// ---------------------------------------------------------------------------
// Procurement tenant config
// ---------------------------------------------------------------------------

// getProcurementConfig returns the procurement configuration for the current tenant.
//
// GET /procurement/config
func (h *ProcurementHandler) getProcurementConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var cfg models.ProcurementTenantConfig
	if err := h.db.ProcurementTenantConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
		// Return defaults when no config document exists yet
		cfg = models.ProcurementTenantConfig{TenantID: tenantID, EkDbMethode: "fifo"}
	}
	writeJSON(w, http.StatusOK, cfg)
}

// updateProcurementConfig updates the procurement configuration for the current tenant.
//
// PUT /procurement/config
func (h *ProcurementHandler) updateProcurementConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		EkDbMethode string `json:"ekDbMethode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	valid := map[string]bool{"last": true, "fifo": true, "lifo": true, "weighted_avg": true}
	if !valid[req.EkDbMethode] {
		http.Error(w, "invalid method: must be last, fifo, lifo, or weighted_avg", http.StatusBadRequest)
		return
	}
	_, err := h.db.ProcurementTenantConfigs().UpdateOne(r.Context(),
		bson.M{"tenantId": tenantID},
		bson.M{"$set": bson.M{"ekDbMethode": req.EkDbMethode, "updatedAt": time.Now()}},
		options.Update().SetUpsert(true))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ekDbMethode": req.EkDbMethode})
}

// ---------------------------------------------------------------------------
// Warehouses (Lager)
// GET    /warehouses          ?q=  &country=  &active=  &page=  &limit=
// POST   /warehouses
// GET    /warehouses/{id}
// PUT    /warehouses/{id}
// DELETE /warehouses/{id}
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listWarehouses(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	andClauses := bson.A{}
	if q := r.URL.Query().Get("q"); q != "" {
		andClauses = append(andClauses, bson.M{"$or": bson.A{
			bson.M{"name": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"shortName": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"description": bson.M{"$regex": q, "$options": "i"}},
		}})
	}
	if country := r.URL.Query().Get("country"); country != "" {
		andClauses = append(andClauses, bson.M{"address.country": bson.M{"$regex": "^" + country + "$", "$options": "i"}})
	}
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		filter["active"] = activeStr == "true"
	}
	if len(andClauses) > 0 {
		filter["$and"] = andClauses
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.Warehouses().CountDocuments(r.Context(), filter)
	cursor, err := h.db.Warehouses().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	items := make([]models.Warehouse, 0)
	cursor.All(r.Context(), &items) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createWarehouse(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.Warehouse
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.Warehouses().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getWarehouse(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Warehouse
	if err := h.db.Warehouses().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateWarehouse(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.Warehouse
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	_, err := h.db.Warehouses().UpdateOne(r.Context(),
		bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"name":        doc.Name,
			"shortName":   doc.ShortName,
			"description": doc.Description,
			"address":     doc.Address,
			"active":      doc.Active,
			"updatedAt":   time.Now().UTC(),
		}},
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	doc.ID = id
	doc.TenantID = tenantID
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) deleteWarehouse(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	_, err := h.db.Warehouses().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Storage Locations (Lagerplätze)
// GET    /storage-locations   ?q=  &warehouseId=  &active=  &page=  &limit=
// POST   /storage-locations
// GET    /storage-locations/{id}
// PUT    /storage-locations/{id}
// DELETE /storage-locations/{id}
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listStorageLocations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	andClauses := bson.A{}
	if q := r.URL.Query().Get("q"); q != "" {
		andClauses = append(andClauses, bson.M{"$or": bson.A{
			bson.M{"name": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"aisle": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"rack": bson.M{"$regex": q, "$options": "i"}},
		}})
	}
	if wid := r.URL.Query().Get("warehouseId"); wid != "" {
		if oid, err := primitive.ObjectIDFromHex(wid); err == nil {
			filter["warehouseId"] = oid
		}
	}
	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		filter["active"] = activeStr == "true"
	}
	if len(andClauses) > 0 {
		filter["$and"] = andClauses
	}
	page, limit, skip := parsePagination(r)
	total, _ := h.db.StorageLocations().CountDocuments(r.Context(), filter)
	cursor, err := h.db.StorageLocations().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	items := make([]models.StorageLocation, 0)
	cursor.All(r.Context(), &items) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) createStorageLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var doc models.StorageLocation
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	doc.ID = primitive.NewObjectID()
	doc.TenantID = tenantID
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = doc.CreatedAt
	if _, err := h.db.StorageLocations().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *ProcurementHandler) getStorageLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.StorageLocation
	if err := h.db.StorageLocations().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&doc); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) updateStorageLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var doc models.StorageLocation
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	_, err := h.db.StorageLocations().UpdateOne(r.Context(),
		bson.M{"_id": id, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"name":        doc.Name,
			"warehouseId": doc.WarehouseID,
			"aisle":       doc.Aisle,
			"rack":        doc.Rack,
			"level":       doc.Level,
			"active":      doc.Active,
			"updatedAt":   time.Now().UTC(),
		}},
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	doc.ID = id
	doc.TenantID = tenantID
	writeJSON(w, http.StatusOK, doc)
}

func (h *ProcurementHandler) deleteStorageLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	_, err := h.db.StorageLocations().DeleteOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID})
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Sales Orders (Verkaufsaufträge – synced from Xentral, read-only)
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listSalesOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"xentralDocumentNr": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"externalOrderNr": bson.M{"$regex": q, "$options": "i"}},
		}
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	page, limit, skip := parsePagination(r)

	sortKey := "date"
	sortDir := int32(-1)

	total, _ := h.db.SalesOrders().CountDocuments(r.Context(), filter)
	cursor, err := h.db.SalesOrders().Find(r.Context(), filter,
		options.Find().
			SetSort(bson.D{{Key: sortKey, Value: sortDir}}).
			SetSkip(skip).SetLimit(limit))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	results := make([]models.SalesOrder, 0)
	cursor.All(r.Context(), &results) //nolint
	writeJSON(w, http.StatusOK, map[string]any{
		"items": results,
		"total": total,
		"page":  page,
		"pages": (total + limit - 1) / limit,
	})
}

func (h *ProcurementHandler) getSalesOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var so models.SalesOrder
	if err := h.db.SalesOrders().FindOne(r.Context(), bson.M{"_id": id, "tenantId": tenantID}).Decode(&so); err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, so)
}

// ---------------------------------------------------------------------------
// Supplier Product Configs (Lead Time, MOQ)
// ---------------------------------------------------------------------------

func (h *ProcurementHandler) listSupplierProductConfigs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	filter := bson.M{"tenantId": tenantID}
	if sid := r.URL.Query().Get("supplierId"); sid != "" {
		if oid, err := primitive.ObjectIDFromHex(sid); err == nil {
			filter["supplierId"] = oid
		}
	}
	if pid := r.URL.Query().Get("productId"); pid != "" {
		if oid, err := primitive.ObjectIDFromHex(pid); err == nil {
			filter["productId"] = oid
		}
	}
	cursor, err := h.db.SupplierProductConfigs().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}}).SetLimit(500),
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var configs []models.SupplierProductConfig
	cursor.All(r.Context(), &configs) //nolint
	if configs == nil {
		configs = []models.SupplierProductConfig{}
	}
	writeJSON(w, http.StatusOK, configs)
}

func (h *ProcurementHandler) upsertSupplierProductConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		SupplierID   string `json:"supplierId"`
		ProductID    string `json:"productId"`
		LeadTimeDays int    `json:"leadTimeDays"`
		MOQ          int    `json:"moq"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	supplierID, err := primitive.ObjectIDFromHex(req.SupplierID)
	if err != nil {
		http.Error(w, "invalid supplierId", http.StatusBadRequest)
		return
	}
	productID, err2 := primitive.ObjectIDFromHex(req.ProductID)
	if err2 != nil {
		http.Error(w, "invalid productId", http.StatusBadRequest)
		return
	}
	if req.LeadTimeDays < 0 || req.MOQ < 0 {
		http.Error(w, "leadTimeDays and moq must be >= 0", http.StatusBadRequest)
		return
	}
	now := time.Now()
	_, upsertErr := h.db.SupplierProductConfigs().UpdateOne(r.Context(),
		bson.M{"tenantId": tenantID, "supplierId": supplierID, "productId": productID},
		bson.M{
			"$set": bson.M{
				"leadTimeDays": req.LeadTimeDays,
				"moq":          req.MOQ,
				"notes":        req.Notes,
				"updatedAt":    now,
			},
			"$setOnInsert": bson.M{
				"tenantId":   tenantID,
				"supplierId": supplierID,
				"productId":  productID,
				"createdAt":  now,
			},
		},
		options.Update().SetUpsert(true),
	)
	if upsertErr != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ProcurementHandler) deleteSupplierProductConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID, err := primitive.ObjectIDFromHex(mux.Vars(r)["supplierId"])
	if err != nil {
		http.Error(w, "invalid supplierId", http.StatusBadRequest)
		return
	}
	productID, err2 := primitive.ObjectIDFromHex(mux.Vars(r)["productId"])
	if err2 != nil {
		http.Error(w, "invalid productId", http.StatusBadRequest)
		return
	}
	h.db.SupplierProductConfigs().DeleteOne(r.Context(), bson.M{ //nolint
		"tenantId": tenantID, "supplierId": supplierID, "productId": productID,
	})
	w.WriteHeader(http.StatusNoContent)
}
