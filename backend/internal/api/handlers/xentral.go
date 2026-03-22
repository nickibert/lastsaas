package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
		BaseURL       string `json:"baseUrl"`
		APIToken      string `json:"apiToken"` // empty = keep existing token
		Enabled       bool   `json:"enabled"`
		SyncIntervalH int    `json:"syncIntervalH"`
		SyncProducts  bool   `json:"syncProducts"`
		SyncCustomers bool   `json:"syncCustomers"`
		SyncSuppliers bool   `json:"syncSuppliers"`
		SyncOrders    bool   `json:"syncOrders"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	now := time.Now()
	set := bson.M{
		"baseUrl":       req.BaseURL,
		"enabled":       req.Enabled,
		"syncIntervalH": req.SyncIntervalH,
		"syncProducts":  req.SyncProducts,
		"syncCustomers": req.SyncCustomers,
		"syncSuppliers": req.SyncSuppliers,
		"syncOrders":    req.SyncOrders,
		"updatedAt":     now,
	}

	// Only update the token if a new one is supplied
	if req.APIToken != "" {
		set["apiToken"] = req.APIToken
		mask := "****"
		if len(req.APIToken) >= 4 {
			mask = "****" + req.APIToken[len(req.APIToken)-4:]
		}
		set["apiTokenMask"] = mask
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

	log := h.runSync(r, tenantID, entity, cfg, client)
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

	var logs []models.XentralSyncLog
	for _, entity := range []string{"products", "customers", "suppliers", "orders"} {
		log := h.runSync(r, tenantID, entity, cfg, client)
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

func (h *XentralHandler) runSync(r *http.Request, tenantID primitive.ObjectID, entity string, cfg models.XentralConfig, client *xentral.Client) models.XentralSyncLog {
	now := time.Now()
	var log models.XentralSyncLog

	switch entity {
	case "products":
		if !cfg.SyncProducts {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncProducts(r.Context(), tenantID, client)
		h.db.XentralConfigs().UpdateOne(r.Context(), bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncProducts": now}}) //nolint
	case "customers":
		if !cfg.SyncCustomers {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncCustomers(r.Context(), tenantID, client)
		h.db.XentralConfigs().UpdateOne(r.Context(), bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncCustomers": now}}) //nolint
	case "suppliers":
		if !cfg.SyncSuppliers {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncSuppliers(r.Context(), tenantID, client)
		h.db.XentralConfigs().UpdateOne(r.Context(), bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncSuppliers": now}}) //nolint
	case "orders":
		if !cfg.SyncOrders {
			return skippedLog(tenantID, entity)
		}
		log = h.engine.SyncOrders(r.Context(), tenantID, client)
		h.db.XentralConfigs().UpdateOne(r.Context(), bson.M{"tenantId": tenantID}, bson.M{"$set": bson.M{"lastSyncOrders": now}}) //nolint
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

func errorf(msg string) error {
	return &simpleErr{msg}
}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
