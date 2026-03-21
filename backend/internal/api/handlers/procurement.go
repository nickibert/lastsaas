package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"lastsaas/internal/db"
	"lastsaas/internal/middleware"
	"lastsaas/internal/models"
	"lastsaas/internal/syslog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/gorilla/mux"
)

type ProcurementHandler struct {
	db             *db.MongoDB
	syslog         *syslog.Logger
	tenantMW       *middleware.TenantMiddleware
}

func NewProcurementHandler(database *db.MongoDB, sysLogger *syslog.Logger) *ProcurementHandler {
	return &ProcurementHandler{
		db:       database,
		syslog:   sysLogger,
		tenantMW: middleware.NewTenantMiddleware(database),
	}
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
	s.HandleFunc("/countries/{id}", h.updateCountry).Methods(http.MethodPut)
	s.HandleFunc("/countries/{id}", h.deleteCountry).Methods(http.MethodDelete)

	// Orders
	s.HandleFunc("/orders", h.listOrders).Methods(http.MethodGet)
	s.HandleFunc("/orders", h.createOrder).Methods(http.MethodPost)
	s.HandleFunc("/orders/{id}", h.getOrder).Methods(http.MethodGet)
	s.HandleFunc("/orders/{id}", h.updateOrder).Methods(http.MethodPut)
	s.HandleFunc("/orders/{id}", h.deleteOrder).Methods(http.MethodDelete)

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
	if q := r.URL.Query().Get("q"); q != "" {
		filter["$or"] = bson.A{
			bson.M{"nameShort": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"ownNameShort": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"nameLong": bson.M{"$regex": q, "$options": "i"}},
			bson.M{"ean": bson.M{"$regex": q, "$options": "i"}},
		}
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
		bson.M{"$set": bson.M{"name": doc.Name, "updatedAt": time.Now().UTC()}}) //nolint
	w.WriteHeader(http.StatusNoContent)
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
	if _, err := h.db.Orders().InsertOne(r.Context(), doc); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
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
		bson.M{"$set": bson.M{"text": doc.Text, "dueDate": doc.DueDate, "doneAt": doc.DoneAt,
			"doneById": doc.DoneByID, "assigneeId": doc.AssigneeID, "updatedAt": doc.UpdatedAt}}) //nolint
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
