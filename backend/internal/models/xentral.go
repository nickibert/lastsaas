package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// XentralConfig holds per-tenant connection settings for the Xentral ERP integration.
// APIToken is stored server-side only and never serialised to JSON (json:"-").
type XentralConfig struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	BaseURL      string             `json:"baseUrl" bson:"baseUrl"`         // e.g. https://mycompany.xentral.biz
	APIToken     string             `json:"-" bson:"apiToken"`              // never sent to client
	APITokenMask string             `json:"apiTokenMask" bson:"apiTokenMask"` // last 4 chars for display
	Enabled      bool               `json:"enabled" bson:"enabled"`

	// Sync interval in hours; 0 = manual only
	SyncIntervalH int `json:"syncIntervalH" bson:"syncIntervalH"`

	// Entity toggles
	SyncProducts  bool `json:"syncProducts" bson:"syncProducts"`
	SyncCustomers bool `json:"syncCustomers" bson:"syncCustomers"`
	SyncSuppliers bool `json:"syncSuppliers" bson:"syncSuppliers"`
	SyncOrders    bool `json:"syncOrders" bson:"syncOrders"`

	// Last successful sync timestamps per entity
	LastSyncProducts  *time.Time `json:"lastSyncProducts,omitempty" bson:"lastSyncProducts,omitempty"`
	LastSyncCustomers *time.Time `json:"lastSyncCustomers,omitempty" bson:"lastSyncCustomers,omitempty"`
	LastSyncSuppliers *time.Time `json:"lastSyncSuppliers,omitempty" bson:"lastSyncSuppliers,omitempty"`
	LastSyncOrders    *time.Time `json:"lastSyncOrders,omitempty" bson:"lastSyncOrders,omitempty"`

	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// XentralMapping links a local record to its Xentral counterpart.
// Used for upsert logic during sync so we avoid duplicate inserts.
type XentralMapping struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId"`
	Entity    string             `json:"entity" bson:"entity"` // "product"|"customer"|"supplier"|"order"
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
