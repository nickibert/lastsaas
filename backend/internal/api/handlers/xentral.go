package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lastsaas/internal/db"
	"lastsaas/internal/models"
	"lastsaas/internal/syslog"
	"lastsaas/internal/xentral"
)

// XentralHandler manages the Xentral ERP integration endpoints.
// All routes sit under /api/procurement/integrations/xentral and use the
// same tenant-isolation pattern as the rest of the procurement module.
type XentralHandler struct {
	db     *db.MongoDB
	syslog *syslog.Logger
	engine *xentral.Engine
}

func NewXentralHandler(database *db.MongoDB, logger *syslog.Logger) *XentralHandler {
	return &XentralHandler{
		db:     database,
		syslog: logger,
		engine: xentral.NewEngine(database),
	}
}

// RegisterRoutes mounts all Xentral routes on s (already scoped to /api/procurement).
func (h *XentralHandler) RegisterRoutes(s *mux.Router) {
	ix := s.PathPrefix("/integrations/xentral").Subrouter()
	ix.HandleFunc("", h.getConfig).Methods(http.MethodGet)
	ix.HandleFunc("", h.saveConfig).Methods(http.MethodPut)
	ix.HandleFunc("/test", h.testConnection).Methods(http.MethodPost)
	ix.HandleFunc("/sync/{entity}", h.triggerSync).Methods(http.MethodPost)
	ix.HandleFunc("/sync", h.triggerSyncAll).Methods(http.MethodPost)
	ix.HandleFunc("/logs", h.listLogs).Methods(http.MethodGet)
	ix.HandleFunc("/mappings", h.listMappings).Methods(http.MethodGet)
	ix.HandleFunc("/import-account", h.importAccount).Methods(http.MethodPost)
	// Bestellvorschläge (purchase suggestions)
	ix.HandleFunc("/purchase-suggestions", h.listPurchaseSuggestions).Methods(http.MethodGet)
	ix.HandleFunc("/purchase-suggestions/generate", h.generatePurchaseSuggestions).Methods(http.MethodPost)
	// Outbound purchase price push
	ix.HandleFunc("/push-purchase-price/{priceId}", h.pushPurchasePrice).Methods(http.MethodPost)
	// Webhook receiver – authenticated by token embedded in the URL.
	// Register on the parent router (no tenant auth middleware) so Xentral can call it.
	s.HandleFunc("/integrations/xentral/webhook/{token}", h.receiveWebhook).Methods(http.MethodPost)
}

// generateWebhookToken creates a 32-byte (64 hex chars) random token.
func generateWebhookToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ---------------------------------------------------------------------------
// GET /integrations/xentral
// ---------------------------------------------------------------------------

func (h *XentralHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg models.XentralConfig
	if err := h.db.XentralConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
		if err == mongo.ErrNoDocuments {
			// Return empty defaults
			writeJSON(w, http.StatusOK, models.XentralConfig{TenantID: tenantID})
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	// APIToken is json:"-" so it's never sent; APITokenMask is sent instead.
	writeJSON(w, http.StatusOK, cfg)
}

// ---------------------------------------------------------------------------
// PUT /integrations/xentral
// ---------------------------------------------------------------------------

func (h *XentralHandler) saveConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		BaseURL               string `json:"baseUrl"`
		APIToken              string `json:"apiToken"` // empty = keep existing token
		Enabled               bool   `json:"enabled"`
		SyncIntervalH         int    `json:"syncIntervalH"`
		SyncProducts          bool   `json:"syncProducts"`
		SyncCustomers         bool   `json:"syncCustomers"`
		SyncSuppliers         bool   `json:"syncSuppliers"`
		SyncOrders            bool   `json:"syncOrders"`
		SyncSalesOrders       bool   `json:"syncSalesOrders"`
		SyncPurchasePrices    bool   `json:"syncPurchasePrices"`
		SyncSalesPrices       bool   `json:"syncSalesPrices"`
		PushProductsToXentral bool   `json:"pushProductsToXentral"`
		PushOrdersToXentral   bool   `json:"pushOrdersToXentral"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	now := time.Now()
	set := bson.M{
		"baseUrl":               req.BaseURL,
		"enabled":               req.Enabled,
		"syncIntervalH":         req.SyncIntervalH,
		"syncProducts":          req.SyncProducts,
		"syncCustomers":         req.SyncCustomers,
		"syncSuppliers":         req.SyncSuppliers,
		"syncOrders":            req.SyncOrders,
		"syncSalesOrders":       req.SyncSalesOrders,
		"syncPurchasePrices":    req.SyncPurchasePrices,
		"syncSalesPrices":       req.SyncSalesPrices,
		"pushProductsToXentral": req.PushProductsToXentral,
		"pushOrdersToXentral":   req.PushOrdersToXentral,
		"updatedAt":             now,
	}

	// Only update the API token if a new one is supplied
	if req.APIToken != "" {
		set["apiToken"] = req.APIToken
		mask := "****"
		if len(req.APIToken) >= 4 {
			mask = "****" + req.APIToken[len(req.APIToken)-4:]
		}
		set["apiTokenMask"] = mask
	}

	// Auto-generate a webhook token on first save if none exists yet.
	var existing models.XentralConfig
	_ = h.db.XentralConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&existing)
	if existing.WebhookToken == "" {
		if token, err := generateWebhookToken(); err == nil {
			set["webhookToken"] = token
		}
	}

	_, err := h.db.XentralConfigs().UpdateOne(r.Context(),
		bson.M{"tenantId": tenantID},
		bson.M{"$set": set},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	h.syslog.Log(r.Context(), "medium", "xentral config updated")
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/test
// ---------------------------------------------------------------------------

func (h *XentralHandler) testConnection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg models.XentralConfig
	if err := h.db.XentralConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
		http.Error(w, "no config saved", http.StatusBadRequest)
		return
	}
	if cfg.BaseURL == "" || cfg.APIToken == "" {
		http.Error(w, "baseUrl and apiToken required", http.StatusBadRequest)
		return
	}

	client := xentral.NewClient(cfg.BaseURL, cfg.APIToken)
	if err := client.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// Fetch instance info from /api/settings (best-effort, non-fatal).
	var instanceInfo *models.XentralInstanceInfo
	if s, err := client.GetSettings(r.Context()); err == nil {
		now := time.Now()
		instanceInfo = &models.XentralInstanceInfo{
			CompanyName: s.ResolvedCompanyName(),
			Version:     s.Version,
			Edition:     s.Edition,
			Email:       s.Email,
			Phone:       s.Phone,
			Website:     s.Website,
			Language:    s.Language,
			Currency:    s.Currency,
			Timezone:    s.Timezone,
			TaxID:       s.TaxID,
			VATID:       s.VATID,
			Street:      s.Street,
			ZIP:         s.ZIP,
			City:        s.City,
			Country:     s.Country,
			FetchedAt:   &now,
		}
		// Persist into config document so the frontend can read it without re-fetching.
		h.db.XentralConfigs().UpdateOne(r.Context(), //nolint
			bson.M{"tenantId": tenantID},
			bson.M{"$set": bson.M{"instanceInfo": instanceInfo}},
		)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "instanceInfo": instanceInfo})
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/sync/{entity}
// entity: products | customers | suppliers | orders
// ---------------------------------------------------------------------------

func (h *XentralHandler) triggerSync(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	entity := mux.Vars(r)["entity"]

	cfg, client, err := h.loadClient(r, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use a detached context so the sync is not cancelled when the HTTP
	// connection drops. Products require one detail API call per item (N+1),
	// so allow up to 8h; all other entities get 2h.
	timeout := 2 * time.Hour
	if entity == "products" {
		timeout = 8 * time.Hour
	}
	syncCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log := h.runSync(syncCtx, tenantID, entity, cfg, client)
	writeJSON(w, http.StatusOK, log)
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/sync  (all enabled entities)
// ---------------------------------------------------------------------------

func (h *XentralHandler) triggerSyncAll(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	cfg, client, err := h.loadClient(r, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Sync order is determined by data dependencies:
	//
	// Phase 1 – Master data (no local dependencies):
	//   suppliers   → referenced by orders
	//   customers   → referenced by sales orders
	//   products    → referenced by prices, stocks, orders
	//   warehouses  → referenced by stocks_per_product
	//
	// Phase 2 – Dependent data (requires phase 1 to be mapped first):
	//   purchase_prices   → needs products
	//   sales_prices      → needs products
	//   orders            → needs products + suppliers
	//   sales_orders      → needs products + customers
	//   stocks_per_product → needs products + warehouses (official v1-beta endpoint)
	//
	// Phase 3 – Outbound (push our data back to Xentral):
	//   push_orders       → pushes local orders into Xentral
	syncOrder := []string{
		// Phase 1
		"merchandise_groups", // must be first so products can link to them
		"suppliers",
		"customers",
		"products",
		"warehouses",
		// Phase 2
		"purchase_prices",
		"sales_prices",
		"orders",
		"sales_orders",
		"stocks_per_product",
		// Phase 3
		"push_orders",
	}

	// Per-entity timeouts: products require one detail API call per item (N+1),
	// which can take several hours for large catalogs. Each entity gets its own
	// independent context so a slow sync does not cancel subsequent entities.
	entityTimeout := map[string]time.Duration{
		"products": 6 * time.Hour,
	}
	defaultTimeout := 2 * time.Hour

	var logs []models.XentralSyncLog
	for _, entity := range syncOrder {
		timeout, ok := entityTimeout[entity]
		if !ok {
			timeout = defaultTimeout
		}
		entityCtx, entityCancel := context.WithTimeout(context.Background(), timeout)
		log := h.runSync(entityCtx, tenantID, entity, cfg, client)
		entityCancel()
		logs = append(logs, log)
	}
	writeJSON(w, http.StatusOK, logs)
}

// ---------------------------------------------------------------------------
// GET /integrations/xentral/logs
// ---------------------------------------------------------------------------

func (h *XentralHandler) listLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	cursor, err := h.db.XentralSyncLogs().Find(r.Context(),
		bson.M{"tenantId": tenantID},
		options.Find().SetSort(bson.D{{Key: "startedAt", Value: -1}}).SetLimit(100),
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var logs []models.XentralSyncLog
	cursor.All(r.Context(), &logs) //nolint
	if logs == nil {
		logs = []models.XentralSyncLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}

// ---------------------------------------------------------------------------
// GET /integrations/xentral/mappings
// ---------------------------------------------------------------------------

func (h *XentralHandler) listMappings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	entity := r.URL.Query().Get("entity")
	filter := bson.M{"tenantId": tenantID}
	if entity != "" {
		filter["entity"] = entity
	}

	cursor, err := h.db.XentralMappings().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(500),
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var mappings []models.XentralMapping
	cursor.All(r.Context(), &mappings) //nolint
	if mappings == nil {
		mappings = []models.XentralMapping{}
	}
	writeJSON(w, http.StatusOK, mappings)
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/import-account
//
// Imports the Xentral instance's own company data into the tenant's
// CompanyProfile (stored in ProcurementTenantConfig). The account holder of
// the Xentral instance is the buyer/tenant — not a supplier — so this data
// belongs in the tenant's own company profile, not in the supplier list.
// ---------------------------------------------------------------------------

func (h *XentralHandler) importAccount(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var cfg models.XentralConfig
	if err := h.db.XentralConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
		http.Error(w, "Xentral nicht konfiguriert", http.StatusBadRequest)
		return
	}
	info := cfg.InstanceInfo
	if info == nil || info.CompanyName == "" {
		// Try to fetch live if not yet cached
		client := xentral.NewClient(cfg.BaseURL, cfg.APIToken)
		s, err := client.GetSettings(r.Context())
		if err != nil {
			http.Error(w, "Konnte Firmendaten nicht abrufen: "+err.Error(), http.StatusBadGateway)
			return
		}
		now := time.Now()
		info = &models.XentralInstanceInfo{
			CompanyName: s.ResolvedCompanyName(),
			Version:     s.Version,
			Edition:     s.Edition,
			Email:       s.Email,
			Phone:       s.Phone,
			Website:     s.Website,
			Language:    s.Language,
			Currency:    s.Currency,
			Timezone:    s.Timezone,
			TaxID:       s.TaxID,
			VATID:       s.VATID,
			Street:      s.Street,
			ZIP:         s.ZIP,
			City:        s.City,
			Country:     s.Country,
			FetchedAt:   &now,
		}
	}
	if info.CompanyName == "" {
		http.Error(w, "Firmenname in Xentral-Stammdaten leer", http.StatusUnprocessableEntity)
		return
	}

	profile := models.CompanyProfile{
		Name:     xentral.Truncate(info.CompanyName, 200),
		Street:   xentral.Truncate(info.Street, 200),
		ZIP:      xentral.Truncate(info.ZIP, 20),
		City:     xentral.Truncate(info.City, 100),
		Country:  xentral.Truncate(info.Country, 50),
		Email:    xentral.Truncate(info.Email, 120),
		Phone:    xentral.Truncate(info.Phone, 50),
		Website:  xentral.Truncate(info.Website, 200),
		TaxID:    xentral.Truncate(info.TaxID, 50),
		VATID:    xentral.Truncate(info.VATID, 50),
		Currency: xentral.Truncate(info.Currency, 10),
		Language: xentral.Truncate(info.Language, 10),
	}

	now := time.Now()
	_, err := h.db.ProcurementTenantConfigs().UpdateOne(r.Context(),
		bson.M{"tenantId": tenantID},
		bson.M{
			"$set": bson.M{
				"companyProfile": profile,
				"updatedAt":      now,
			},
			"$setOnInsert": bson.M{
				"tenantId":    tenantID,
				"ekDbMethode": "last",
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	h.syslog.Log(r.Context(), "medium", "xentral company profile imported: "+info.CompanyName)
	writeJSON(w, http.StatusOK, map[string]any{"companyProfile": profile})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (h *XentralHandler) loadClient(r *http.Request, tenantID primitive.ObjectID) (models.XentralConfig, *xentral.Client, error) {
	var cfg models.XentralConfig
	if err := h.db.XentralConfigs().FindOne(r.Context(), bson.M{"tenantId": tenantID}).Decode(&cfg); err != nil {
		return cfg, nil, errorf("Xentral nicht konfiguriert")
	}
	if !cfg.Enabled {
		return cfg, nil, errorf("Xentral-Integration ist deaktiviert")
	}
	if cfg.BaseURL == "" || cfg.APIToken == "" {
		return cfg, nil, errorf("baseUrl und apiToken erforderlich")
	}
	return cfg, xentral.NewClient(cfg.BaseURL, cfg.APIToken), nil
}

func (h *XentralHandler) runSync(ctx context.Context, tenantID primitive.ObjectID, entity string, cfg models.XentralConfig, client *xentral.Client) models.XentralSyncLog {
	now := time.Now()
	var log models.XentralSyncLog

	switch entity {
	case "merchandise_groups":
		if !cfg.SyncProducts { // reuse SyncProducts flag as gate for merchandise groups
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncMerchandiseGroups(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncMerchandiseGroups": now}}) //nolint
	case "products":
		if !cfg.SyncProducts {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncProducts(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncProducts": now}}) //nolint
	case "customers":
		if !cfg.SyncCustomers {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncCustomers(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncCustomers": now}}) //nolint
	case "suppliers":
		if !cfg.SyncSuppliers {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncSuppliers(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncSuppliers": now}}) //nolint
	case "orders":
		// "orders" = one-time import FROM Xentral (migration). For ongoing sync use "push_orders".
		if !cfg.SyncOrders {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncOrders(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncOrders": now}}) //nolint
	case "push_orders":
		if !cfg.PushOrdersToXentral {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.PushAllOrdersToXentral(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastPushOrders": now}}) //nolint
	case "sales_orders":
		if !cfg.SyncSalesOrders {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncSalesOrders(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncSalesOrders": now}}) //nolint
	case "purchase_prices":
		if !cfg.SyncPurchasePrices {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncPurchasePrices(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncPurchasePrices": now}}) //nolint
	case "sales_prices":
		if !cfg.SyncSalesPrices {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncSalesPrices(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncSalesPrices": now}}) //nolint
	case "warehouses":
		if !cfg.SyncWarehouses {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncWarehouses(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncWarehouses": now}}) //nolint
	case "stocks":
		if !cfg.SyncStocks {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncStocks(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncStocks": now}}) //nolint
	case "stocks_per_product":
		// Uses the official /api/products/:id/stocks (v1-beta) endpoint.
		// Intended for the initial stock import; requires warehouses to be synced first.
		if !cfg.SyncStocks {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncStocksPerProduct(ctx, tenantID, client)
		h.db.XentralConfigs().UpdateOne(ctx, bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncStocks": now}}) //nolint
	default:
		return skippedLog(tenantID, entity)
	}
	return log
}


func skippedLog(tenantID primitive.ObjectID, entity string) models.XentralSyncLog {
	now := time.Now()
	return models.XentralSyncLog{
		TenantID:   tenantID,
		Entity:     entity,
		Status:     "skipped",
		StartedAt:  now,
		FinishedAt: &now,
	}
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/webhook/{token}
// Xentral calls this URL whenever an entity changes (product, customer, etc.).
// The token in the URL authenticates the call (set in Xentral: Settings → Webhooks).
// ---------------------------------------------------------------------------

// xentralWebhookPayload is the payload Xentral sends for each webhook event.
// Xentral sends: {"event": "product.updated", "resourceId": "42", ...}
type xentralWebhookPayload struct {
	Event      string `json:"event"`
	ResourceID string `json:"resourceId"`
}

func (h *XentralHandler) receiveWebhook(w http.ResponseWriter, r *http.Request) {
	token := mux.Vars(r)["token"]
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	// Find the tenant config that owns this webhook token.
	var cfg models.XentralConfig
	if err := h.db.XentralConfigs().FindOne(r.Context(),
		bson.M{"webhookToken": token, "enabled": true},
	).Decode(&cfg); err != nil {
		// Return 200 to avoid Xentral retrying with an invalid token.
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload xentralWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	client := xentral.NewClient(cfg.BaseURL, cfg.APIToken)
	tenantID := cfg.TenantID
	resourceID := payload.ResourceID

	// Targeted incremental sync: use resourceId when present to sync only the changed
	// entity. If resourceId is missing fall back to a full entity list sync.
	// Run in a goroutine so the HTTP response is returned immediately (Xentral expects
	// a fast 200 acknowledgement; our detail fetches can take seconds).
	go func() {
		wCtx, wCancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer wCancel()
		switch {
		case isEventType(payload.Event, "product"):
			if resourceID != "" {
				h.engine.SyncSingleProduct(wCtx, tenantID, client, resourceID) //nolint
			} else {
				h.engine.SyncProducts(wCtx, tenantID, client) //nolint
			}
		case isEventType(payload.Event, "customer"):
			if resourceID != "" {
				h.engine.SyncSingleCustomer(wCtx, tenantID, client, resourceID) //nolint
			} else {
				h.engine.SyncCustomers(wCtx, tenantID, client) //nolint
			}
		case isEventType(payload.Event, "supplier"):
			if resourceID != "" {
				h.engine.SyncSingleSupplier(wCtx, tenantID, client, resourceID) //nolint
			} else {
				h.engine.SyncSuppliers(wCtx, tenantID, client) //nolint
			}
		case isEventType(payload.Event, "purchaseOrder"):
			if resourceID != "" {
				h.engine.SyncSinglePurchaseOrder(wCtx, tenantID, client, resourceID) //nolint
			} else {
				h.engine.SyncOrders(wCtx, tenantID, client) //nolint
			}
		case isEventType(payload.Event, "salesOrder"):
			if resourceID != "" {
				h.engine.SyncSingleSalesOrder(wCtx, tenantID, client, resourceID) //nolint
			} else {
				h.engine.SyncSalesOrders(wCtx, tenantID, client) //nolint
			}
		case isEventType(payload.Event, "purchasePrice"):
			h.engine.SyncPurchasePrices(wCtx, tenantID, client) //nolint
		case isEventType(payload.Event, "salesPrice"):
			h.engine.SyncSalesPrices(wCtx, tenantID, client) //nolint
		case isEventType(payload.Event, "merchandiseGroup"):
			h.engine.SyncMerchandiseGroups(wCtx, tenantID, client) //nolint
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// isEventType checks whether a Xentral event string matches an entity prefix.
// Xentral events follow the pattern "<entity>.<verb>", e.g. "product.updated".
func isEventType(event, entity string) bool {
	return len(event) > len(entity) && event[:len(entity)+1] == entity+"."
}

func errorf(msg string) error {
	return &simpleErr{msg}
}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }

// ---------------------------------------------------------------------------
// GET /integrations/xentral/purchase-suggestions
// ---------------------------------------------------------------------------

func (h *XentralHandler) listPurchaseSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filter := bson.M{"tenantId": tenantID}
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	cursor, err := h.db.PurchaseSuggestions().Find(r.Context(), filter,
		options.Find().SetSort(bson.D{{Key: "generatedAt", Value: -1}}).SetLimit(500),
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var suggestions []models.PurchaseSuggestion
	cursor.All(r.Context(), &suggestions) //nolint
	if suggestions == nil {
		suggestions = []models.PurchaseSuggestion{}
	}
	writeJSON(w, http.StatusOK, suggestions)
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/purchase-suggestions/generate
// ---------------------------------------------------------------------------

func (h *XentralHandler) generatePurchaseSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		AnalysisDays int `json:"analysisDays"` // default 90
		TargetDays   int `json:"targetDays"`   // default 60
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.AnalysisDays <= 0 {
		req.AnalysisDays = 90
	}
	if req.TargetDays <= 0 {
		req.TargetDays = 60
	}

	suggestions, err := h.engine.GeneratePurchaseSuggestions(r.Context(), tenantID, req.AnalysisDays, req.TargetDays)
	if err != nil {
		http.Error(w, "generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if suggestions == nil {
		suggestions = []models.PurchaseSuggestion{}
	}
	h.syslog.Log(r.Context(), "low", "purchase suggestions generated")
	writeJSON(w, http.StatusOK, suggestions)
}

// ---------------------------------------------------------------------------
// POST /integrations/xentral/push-purchase-price/{priceId}
// Pushes a single ProductPriceList (EK) entry to Xentral.
// Used when confirming a purchase price after goods arrival or pre-calculating EK.
// ---------------------------------------------------------------------------

func (h *XentralHandler) pushPurchasePrice(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := procurementTenantID(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	priceIDStr := mux.Vars(r)["priceId"]
	priceID, err := primitive.ObjectIDFromHex(priceIDStr)
	if err != nil {
		http.Error(w, "invalid priceId", http.StatusBadRequest)
		return
	}

	_, client, err := h.loadClient(r, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	xentralID, pushErr := h.engine.PushPurchasePriceToXentral(r.Context(), tenantID, priceID, client)
	if pushErr != nil {
		http.Error(w, pushErr.Error(), http.StatusBadGateway)
		return
	}

	h.syslog.Log(r.Context(), "medium", "purchase price pushed to Xentral: "+xentralID)
	writeJSON(w, http.StatusOK, map[string]string{"xentralId": xentralID})
}
