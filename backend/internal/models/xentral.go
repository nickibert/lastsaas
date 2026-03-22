package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// XentralInstanceInfo caches the data returned by the undocumented /api/settings
// endpoint. Populated on every successful connection test.
type XentralInstanceInfo struct {
	CompanyName string `json:"companyName" bson:"companyName"`
	Version     string `json:"version" bson:"version"`
	Edition     string `json:"edition" bson:"edition"`
	Email       string `json:"email" bson:"email"`
	Phone       string `json:"phone" bson:"phone"`
	Website     string `json:"website" bson:"website"`
	Language    string `json:"language" bson:"language"`
	Currency    string `json:"currency" bson:"currency"`
	Timezone    string `json:"timezone" bson:"timezone"`
	TaxID       string `json:"taxId" bson:"taxId"`
	VATID       string `json:"vatId" bson:"vatId"`
	Street      string `json:"street" bson:"street"`
	ZIP         string `json:"zip" bson:"zip"`
	City        string `json:"city" bson:"city"`
	Country     string `json:"country" bson:"country"`
	FetchedAt   *time.Time `json:"fetchedAt,omitempty" bson:"fetchedAt,omitempty"`
}

// XentralConfig holds per-tenant connection settings for the Xentral ERP integration.
// APIToken is stored server-side only and never serialised to JSON (json:"-").
type XentralConfig struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	BaseURL      string             `json:"baseUrl" bson:"baseUrl"`           // e.g. https://mycompany.xentral.biz
	APIToken     string             `json:"-" bson:"apiToken"`                // never sent to client
	APITokenMask string             `json:"apiTokenMask" bson:"apiTokenMask"` // last 4 chars for display
	Enabled      bool               `json:"enabled" bson:"enabled"`

	// Cached info from /api/settings, populated on successful connection test.
	InstanceInfo *XentralInstanceInfo `json:"instanceInfo,omitempty" bson:"instanceInfo,omitempty"`

	// Sync interval in hours; 0 = manual only
	SyncIntervalH int `json:"syncIntervalH" bson:"syncIntervalH"`

	// Entity toggles (inbound: Xentral → LastSaaS)
	SyncProducts        bool `json:"syncProducts" bson:"syncProducts"`
	SyncCustomers       bool `json:"syncCustomers" bson:"syncCustomers"`
	SyncSuppliers       bool `json:"syncSuppliers" bson:"syncSuppliers"`
	// SyncOrders: one-time import of purchase orders FROM Xentral (migration only).
	// For ongoing sync LastSaaS is the SPoT: use PushOrdersToXentral instead.
	SyncOrders          bool `json:"syncOrders" bson:"syncOrders"`
	SyncSalesOrders     bool `json:"syncSalesOrders" bson:"syncSalesOrders"`       // sales orders (reorder planning)
	SyncPurchasePrices  bool `json:"syncPurchasePrices" bson:"syncPurchasePrices"` // EK price lists from Xentral
	SyncSalesPrices     bool `json:"syncSalesPrices" bson:"syncSalesPrices"`       // VK price lists from Xentral

	// Outbound toggles (LastSaaS → Xentral)
	PushProductsToXentral bool `json:"pushProductsToXentral" bson:"pushProductsToXentral"` // EAN + freefields
	// PushOrdersToXentral: LastSaaS is SPoT for purchase orders.
	// On create/update in LastSaaS the PO is created/patched in Xentral.
	PushOrdersToXentral   bool `json:"pushOrdersToXentral" bson:"pushOrdersToXentral"`

	// Last successful sync timestamps per entity
	LastSyncProducts       *time.Time `json:"lastSyncProducts,omitempty" bson:"lastSyncProducts,omitempty"`
	LastSyncCustomers      *time.Time `json:"lastSyncCustomers,omitempty" bson:"lastSyncCustomers,omitempty"`
	LastSyncSuppliers      *time.Time `json:"lastSyncSuppliers,omitempty" bson:"lastSyncSuppliers,omitempty"`
	LastSyncOrders         *time.Time `json:"lastSyncOrders,omitempty" bson:"lastSyncOrders,omitempty"` // last import
	LastPushOrders         *time.Time `json:"lastPushOrders,omitempty" bson:"lastPushOrders,omitempty"` // last push
	LastSyncSalesOrders    *time.Time `json:"lastSyncSalesOrders,omitempty" bson:"lastSyncSalesOrders,omitempty"`
	LastSyncPurchasePrices *time.Time `json:"lastSyncPurchasePrices,omitempty" bson:"lastSyncPurchasePrices,omitempty"`
	LastSyncSalesPrices    *time.Time `json:"lastSyncSalesPrices,omitempty" bson:"lastSyncSalesPrices,omitempty"`

	// WebhookToken is a random token embedded in the Xentral webhook URL so that
	// only Xentral can trigger real-time sync updates.
	// URL pattern: POST /api/procurement/integrations/xentral/webhook/{webhookToken}
	WebhookToken string `json:"webhookToken,omitempty" bson:"webhookToken,omitempty"`

	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// XentralMapping links a local record to its Xentral counterpart.
// Used for upsert logic during sync so we avoid duplicate inserts.
type XentralMapping struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	Entity    string             `json:"entity" bson:"entity"` // "product"|"customer"|"supplier"|"order"|"sales_order"
	LocalID   primitive.ObjectID `json:"localId" bson:"localId"`
	XentralID string             `json:"xentralId" bson:"xentralId"` // Xentral UUID
	XentralNr string             `json:"xentralNr,omitempty" bson:"xentralNr,omitempty"` // e.g. "ART-001"
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// XentralSyncLog records the result of one sync run for one entity type.
type XentralSyncLog struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID   primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	Entity     string             `json:"entity" bson:"entity"` // "products"|"customers"|"suppliers"|"orders"
	Status     string             `json:"status" bson:"status"` // "running"|"success"|"error"|"partial"
	Created    int                `json:"created" bson:"created"`
	Updated    int                `json:"updated" bson:"updated"`
	Skipped    int                `json:"skipped" bson:"skipped"`
	Errors     []string           `json:"errors,omitempty" bson:"errors,omitempty"`
	StartedAt  time.Time          `json:"startedAt" bson:"startedAt"`
	FinishedAt *time.Time         `json:"finishedAt,omitempty" bson:"finishedAt,omitempty"`
}
