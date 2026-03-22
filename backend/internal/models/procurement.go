package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---------------------------------------------------------------------------
// Supplier
// ---------------------------------------------------------------------------

type Supplier struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Company   string             `json:"company" bson:"company" validate:"required,min=1,max=200"`
	Firstname string             `json:"firstname" bson:"firstname" validate:"omitempty,max=50"`
	Lastname  string             `json:"lastname" bson:"lastname" validate:"omitempty,max=50"`
	Email     string             `json:"email" bson:"email" validate:"omitempty,email,max=120"`
	Skype     string             `json:"skype" bson:"skype" validate:"omitempty,max=100"`
	Origin    string             `json:"origin" bson:"origin" validate:"omitempty,max=50"`
	Misc      string             `json:"misc" bson:"misc" validate:"omitempty,max=500"`
	Tags      []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	LegacyID  int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Customer
// ---------------------------------------------------------------------------

type CustomerAddress struct {
	Street  string `json:"street,omitempty" bson:"street,omitempty" validate:"omitempty,max=200"`
	City    string `json:"city,omitempty" bson:"city,omitempty" validate:"omitempty,max=100"`
	Zip     string `json:"zip,omitempty" bson:"zip,omitempty" validate:"omitempty,max=20"`
	Country string `json:"country,omitempty" bson:"country,omitempty" validate:"omitempty,max=100"`
}

type Customer struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Company     string             `json:"company" bson:"company" validate:"required,min=1,max=200"`
	Firstname   string             `json:"firstname,omitempty" bson:"firstname,omitempty" validate:"omitempty,max=50"`
	Lastname    string             `json:"lastname,omitempty" bson:"lastname,omitempty" validate:"omitempty,max=50"`
	Email       string             `json:"email,omitempty" bson:"email,omitempty" validate:"omitempty,email,max=120"`
	Phone       string             `json:"phone,omitempty" bson:"phone,omitempty" validate:"omitempty,max=50"`
	Address     CustomerAddress    `json:"address,omitempty" bson:"address,omitempty"`
	Misc        string             `json:"misc,omitempty" bson:"misc,omitempty" validate:"omitempty,max=1000"`
	Tags        []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type SupplierCode struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID   primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	SupplierID primitive.ObjectID `json:"supplierId" bson:"supplierId" validate:"required"`
	Short      string             `json:"short" bson:"short" validate:"required,min=1,max=10"`
	LegacyID   int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt  time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// GoodsGroup
// ---------------------------------------------------------------------------

type GoodsGroup struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name      string             `json:"name" bson:"name" validate:"required,min=1,max=100"`
	Short     string             `json:"short" bson:"short" validate:"required,min=1,max=10"`
	LegacyID  int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Product
// ---------------------------------------------------------------------------

type ProductAttribute struct {
	Key   string `json:"key" bson:"key" validate:"required,min=1,max=100"`
	Value string `json:"value" bson:"value" validate:"max=500"`
	Unit  string `json:"unit,omitempty" bson:"unit,omitempty" validate:"omitempty,max=30"`
}

type Product struct {
	ID             primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID       primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	NameShort      string              `json:"nameShort" bson:"nameShort" validate:"required,min=1,max=45"`
	OwnNameShort   string              `json:"ownNameShort" bson:"ownNameShort" validate:"omitempty,max=25"`
	NameLong       string              `json:"nameLong" bson:"nameLong" validate:"omitempty,max=255"`
	Description    string              `json:"description" bson:"description"`
	Misc           string              `json:"misc" bson:"misc"`
	WTN            string              `json:"wtn" bson:"wtn" validate:"omitempty,max=30"`
	EAN            string              `json:"ean" bson:"ean" validate:"omitempty,max=32"`
	ACO            string              `json:"aco" bson:"aco" validate:"omitempty,max=3"`
	GoodsGroupID   *primitive.ObjectID `json:"goodsGroupId,omitempty" bson:"goodsGroupId,omitempty"`
	SupplierCodeID *primitive.ObjectID `json:"supplierCodeId,omitempty" bson:"supplierCodeId,omitempty"`
	SupplierID     *primitive.ObjectID `json:"supplierId,omitempty" bson:"supplierId,omitempty"`
	// Package dimensions (stored at product level as default)
	WidthMM  int     `json:"widthMm" bson:"widthMm" validate:"min=0"`
	HeightMM int     `json:"heightMm" bson:"heightMm" validate:"min=0"`
	LengthMM int     `json:"lengthMm" bson:"lengthMm" validate:"min=0"`
	WeightKg float64 `json:"weightKg" bson:"weightKg" validate:"min=0"`
	VPE      int     `json:"vpe" bson:"vpe" validate:"min=0"` // Verpackungseinheit
	// Supplier relationships (many-to-many via legacy products_supplier table)
	SupplierIDs []primitive.ObjectID `json:"supplierIds,omitempty" bson:"supplierIds,omitempty"`
	// Pricing
	LastEK     float64    `json:"lastEk" bson:"lastEk" validate:"min=0"`
	LastEKDate *time.Time `json:"lastEkDate,omitempty" bson:"lastEkDate,omitempty"`
	// PIM: Tags and structured attributes
	Tags       []string           `json:"tags,omitempty" bson:"tags,omitempty"`
	Attributes []ProductAttribute `json:"attributes,omitempty" bson:"attributes,omitempty"`
	// Custom fields (legacy userdef01-10)
	UserDef01 string `json:"userDef01" bson:"userDef01"`
	UserDef02 string `json:"userDef02" bson:"userDef02"`
	UserDef03 string `json:"userDef03" bson:"userDef03"`
	UserDef04 string `json:"userDef04" bson:"userDef04"`
	UserDef05 string `json:"userDef05" bson:"userDef05"`
	UserDef06 string `json:"userDef06" bson:"userDef06"`
	UserDef07 string `json:"userDef07" bson:"userDef07"`
	UserDef08 string `json:"userDef08" bson:"userDef08"`
	UserDef09 string `json:"userDef09" bson:"userDef09"`
	UserDef10 string `json:"userDef10" bson:"userDef10"`
	Checked   bool   `json:"checked" bson:"checked"`
	Virtual   bool   `json:"virtual" bson:"virtual"`
	LegacyID  int    `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type ProductPriceList struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	ProductID    primitive.ObjectID  `json:"productId" bson:"productId" validate:"required"`
	Type         string              `json:"type" bson:"type" validate:"required,oneof=EK VK"`
	Name         string              `json:"name" bson:"name" validate:"required,min=1,max=100"`
	Price        float64             `json:"price" bson:"price" validate:"min=0"`
	Currency     string              `json:"currency" bson:"currency" validate:"required,len=3"`
	ValidFrom    *time.Time          `json:"validFrom,omitempty" bson:"validFrom,omitempty"`
	ValidTo      *time.Time          `json:"validTo,omitempty" bson:"validTo,omitempty"`
	Notes        string              `json:"notes,omitempty" bson:"notes,omitempty" validate:"omitempty,max=500"`
	// Provenance: set when created by applyOrderEK
	OrderID      *primitive.ObjectID `json:"orderId,omitempty" bson:"orderId,omitempty"`
	FreightIndex *int                `json:"freightIndex,omitempty" bson:"freightIndex,omitempty"`
	Quantity     int                 `json:"quantity,omitempty" bson:"quantity,omitempty"`
	Source       string              `json:"source,omitempty" bson:"source,omitempty"` // "order" | "import" | "manual"
	CreatedAt    time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Inventory Lots (FIFO / LIFO / Weighted-Average valuation)
// ---------------------------------------------------------------------------

// InventoryLot records a single goods receipt batch for one product.
// Each container arrival writes a separate lot so that FIFO/LIFO methods
// can walk the lots in chronological order.
type InventoryLot struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	ProductID    primitive.ObjectID  `json:"productId" bson:"productId" validate:"required"`
	OrderID      *primitive.ObjectID `json:"orderId,omitempty" bson:"orderId,omitempty"`
	FreightIndex *int                `json:"freightIndex,omitempty" bson:"freightIndex,omitempty"`
	ReceivedAt   time.Time           `json:"receivedAt" bson:"receivedAt" validate:"required"`
	Quantity     int                 `json:"quantity" bson:"quantity" validate:"required,min=1"`
	// Remaining is decremented on each outgoing stock movement; starts == Quantity.
	Remaining    int                 `json:"remaining" bson:"remaining" validate:"min=0"`
	UnitEkEUR    float64             `json:"unitEkEur" bson:"unitEkEur" validate:"min=0"`
	// Source: "order" | "import" (manual stock take-over)
	Source       string              `json:"source" bson:"source" validate:"required,oneof=order import"`
	Notes        string              `json:"notes,omitempty" bson:"notes,omitempty" validate:"omitempty,max=500"`
	CreatedAt    time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Freight / Logistics
// ---------------------------------------------------------------------------

type FreightCarrier struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name        string             `json:"name" bson:"name" validate:"required,min=1,max=50"`
	Description string             `json:"description" bson:"description" validate:"omitempty,max=255"`
	LegacyID    int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type FreightPrice struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID         primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	FreightCarrierID primitive.ObjectID `json:"freightCarrierId" bson:"freightCarrierId" validate:"required"`
	ContainerID      primitive.ObjectID `json:"containerId" bson:"containerId" validate:"required"`
	HarbourIDFrom    primitive.ObjectID `json:"harbourIdFrom" bson:"harbourIdFrom" validate:"required"`
	HarbourIDTo      primitive.ObjectID `json:"harbourIdTo" bson:"harbourIdTo" validate:"required"`
	PriceEUR         float64            `json:"priceEur" bson:"priceEur" validate:"min=0"`
	ValidFrom        *time.Time         `json:"validFrom,omitempty" bson:"validFrom,omitempty"`
	ValidTo          *time.Time         `json:"validTo,omitempty" bson:"validTo,omitempty"`
	LegacyID         int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt        time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type Harbour struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name        string             `json:"name" bson:"name" validate:"required,min=1,max=50"`
	Description string             `json:"description" bson:"description" validate:"omitempty,max=255"`
	LegacyID    int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type Container struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name        string             `json:"name" bson:"name" validate:"required,min=1,max=50"`
	Description string             `json:"description" bson:"description" validate:"omitempty,max=255"`
	VolumeM3    float64            `json:"volumeM3" bson:"volumeM3" validate:"min=0"`
	HeightM     float64            `json:"heightM" bson:"heightM" validate:"min=0"`
	LengthM     float64            `json:"lengthM" bson:"lengthM" validate:"min=0"`
	WidthM      float64            `json:"widthM" bson:"widthM" validate:"min=0"`
	LegacyID    int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type CustomerFreightCarrier struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name        string             `json:"name" bson:"name" validate:"required,min=1,max=100"`
	Description string             `json:"description" bson:"description" validate:"omitempty,max=255"`
	LegacyID    int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type Country struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID       primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Name           string             `json:"name" bson:"name" validate:"required,min=1,max=100"`
	ISO2           string             `json:"iso2,omitempty" bson:"iso2,omitempty" validate:"omitempty,len=2"`
	ISO3           string             `json:"iso3,omitempty" bson:"iso3,omitempty" validate:"omitempty,len=3"`
	ISONumeric     string             `json:"isoNumeric,omitempty" bson:"isoNumeric,omitempty" validate:"omitempty,max=3"`
	Currency       string             `json:"currency,omitempty" bson:"currency,omitempty" validate:"omitempty,max=100"`
	CurrencyCode   string             `json:"currencyCode,omitempty" bson:"currencyCode,omitempty" validate:"omitempty,max=3"`
	CurrencySymbol string             `json:"currencySymbol,omitempty" bson:"currencySymbol,omitempty" validate:"omitempty,max=10"`
	PhoneCode      string             `json:"phoneCode,omitempty" bson:"phoneCode,omitempty" validate:"omitempty,max=15"`
	Region         string             `json:"region,omitempty" bson:"region,omitempty" validate:"omitempty,max=100"`
	Capital        string             `json:"capital,omitempty" bson:"capital,omitempty" validate:"omitempty,max=100"`
	LegacyID       int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Order
// ---------------------------------------------------------------------------

type OrderPayment struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID          primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	OrderID           primitive.ObjectID `json:"orderId" bson:"orderId" validate:"required"`
	Nr                int                `json:"nr" bson:"nr" validate:"min=0"` // 0=deposit, 1=final
	PaymentAmountEUR  float64            `json:"paymentAmountEur" bson:"paymentAmountEur" validate:"min=0"`
	PaymentDate       *time.Time         `json:"paymentDate,omitempty" bson:"paymentDate,omitempty"`
	PaymentDollarRate float64            `json:"paymentDollarRate" bson:"paymentDollarRate" validate:"min=0"`
	PaymentFees       float64            `json:"paymentFees" bson:"paymentFees" validate:"min=0"`
	LegacyID          int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt         time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type OrderProduct struct {
	ProductID        primitive.ObjectID `json:"productId" bson:"productId" validate:"required"`
	Quantity         int                `json:"quantity" bson:"quantity" validate:"required,min=1"`
	UnitPriceUSD     float64            `json:"unitPriceUsd" bson:"unitPriceUsd" validate:"min=0"`
	TotalPriceUSD    float64            `json:"totalPriceUsd" bson:"totalPriceUsd" validate:"min=0"`
	LengthMM         int                `json:"lengthMm" bson:"lengthMm" validate:"min=0"`
	WidthMM          int                `json:"widthMm" bson:"widthMm" validate:"min=0"`
	HeightMM         int                `json:"heightMm" bson:"heightMm" validate:"min=0"`
	VolumeM3         float64            `json:"volumeM3" bson:"volumeM3" validate:"min=0"`
	WeightKg         float64            `json:"weightKg" bson:"weightKg" validate:"min=0"`
	Credited         bool               `json:"credited" bson:"credited"`
	InventoryChecked bool               `json:"inventoryChecked" bson:"inventoryChecked"`
	// FreightIndex links this line item to freights[i]. nil = costs spread across all containers (legacy).
	FreightIndex *int `json:"freightIndex,omitempty" bson:"freightIndex,omitempty"`
}

type OrderFreight struct {
	ContainerID                 primitive.ObjectID  `json:"containerId" bson:"containerId"`
	HarbourIDFrom               primitive.ObjectID  `json:"harbourIdFrom" bson:"harbourIdFrom"`
	HarbourIDTo                 primitive.ObjectID  `json:"harbourIdTo" bson:"harbourIdTo"`
	FreightCarrierID            primitive.ObjectID  `json:"freightCarrierId" bson:"freightCarrierId"`
	ContainerNr                 string              `json:"containerNr" bson:"containerNr" validate:"omitempty,max=32"`
	ShippingDate                *time.Time          `json:"shippingDate,omitempty" bson:"shippingDate,omitempty"`
	EstimatedArrival            *time.Time          `json:"estimatedArrival,omitempty" bson:"estimatedArrival,omitempty"`
	Arrival                     *time.Time          `json:"arrival,omitempty" bson:"arrival,omitempty"`
	AvisShipperDate             *time.Time          `json:"avisShipperDate,omitempty" bson:"avisShipperDate,omitempty"`
	DocOfOrigin                 *time.Time          `json:"docOfOrigin,omitempty" bson:"docOfOrigin,omitempty"`
	DocOfOriginChecked          bool                `json:"docOfOriginChecked" bson:"docOfOriginChecked"`
	DocOfOriginSigned           bool                `json:"docOfOriginSigned" bson:"docOfOriginSigned"`
	DocOfOriginShipped          bool                `json:"docOfOriginShipped" bson:"docOfOriginShipped"`
	ProformaInvoiceFileID       *primitive.ObjectID `json:"proformaInvoiceFileId,omitempty" bson:"proformaInvoiceFileId,omitempty"`
	CommercialInvoiceFileID     *primitive.ObjectID `json:"commercialInvoiceFileId,omitempty" bson:"commercialInvoiceFileId,omitempty"`
	PackingListFileID           *primitive.ObjectID `json:"packingListFileId,omitempty" bson:"packingListFileId,omitempty"`
	FreighterInvoiceFileID      *primitive.ObjectID `json:"freighterInvoiceFileId,omitempty" bson:"freighterInvoiceFileId,omitempty"`
	// Sea freight costs (USD)
	SeaFreightUSD               float64 `json:"seaFreightUsd" bson:"seaFreightUsd" validate:"min=0"`
	EmergencyBunkerSurchargeUSD float64 `json:"emergencyBunkerSurchargeUsd" bson:"emergencyBunkerSurchargeUsd" validate:"min=0"`
	PeakSeasonSurchargeUSD      float64 `json:"peakSeasonSurchargeUsd" bson:"peakSeasonSurchargeUsd" validate:"min=0"`
	SuezCanalAddonUSD           float64 `json:"suezCanalAddonUsd" bson:"suezCanalAddonUsd" validate:"min=0"`
	DangerPayUSD                float64 `json:"dangerPayUsd" bson:"dangerPayUsd" validate:"min=0"`
	// Dollar rate used for this freight leg (may differ from order's preDollarRate)
	DollarRate                  float64 `json:"dollarRate" bson:"dollarRate" validate:"min=0"`
	// Domestic / port costs (EUR)
	FreightageEUR               float64 `json:"freightageEur" bson:"freightageEur" validate:"min=0"`
	PreFreightageEUR            float64 `json:"preFreightageEur" bson:"preFreightageEur" validate:"min=0"`
	THCEUR                      float64 `json:"thcEur" bson:"thcEur" validate:"min=0"`           // Terminal Handling Charge
	ISPSEUR                     float64 `json:"ispsEur" bson:"ispsEur" validate:"min=0"`          // ISPS Security Surcharge
	BLDocFeeEUR                 float64 `json:"blDocFeeEur" bson:"blDocFeeEur" validate:"min=0"`  // Bill of Lading doc fee
	FollowUpFeesEUR             float64 `json:"followUpFeesEur" bson:"followUpFeesEur" validate:"min=0"`
	// Customs (EUR)
	CustomsClearanceEUR         float64 `json:"customsClearanceEur" bson:"customsClearanceEur" validate:"min=0"`
	CustomsEUR                  float64 `json:"customsEur" bson:"customsEur" validate:"min=0"`
	CustomsPercent              float64 `json:"customsPercent" bson:"customsPercent" validate:"min=0"`
	ZTN                         string  `json:"ztn" bson:"ztn" validate:"omitempty,max=12"` // Zolltarifnummer reference
}

// Order delivery/payment/warehouse statuses
// DeliveryStatus: pending | ordered | shipped | arrived | partial
// PaymentStatus:  unpaid | deposit_paid | fully_paid | overdue
// ReceiptStatus:  pending | partial | received | distributed

type Order struct {
	ID                         primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID                   primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	SupplierID                 *primitive.ObjectID `json:"supplierId,omitempty" bson:"supplierId,omitempty"`
	CustomerID                 *primitive.ObjectID `json:"customerId,omitempty" bson:"customerId,omitempty"`
	OrderNumber                string              `json:"orderNumber" bson:"orderNumber" validate:"omitempty,max=64"`
	InternalNumber             string              `json:"internalNumber,omitempty" bson:"internalNumber,omitempty" validate:"omitempty,max=32"`
	OrderDate                  time.Time           `json:"orderDate" bson:"orderDate" validate:"required"`
	OrderContents              string              `json:"orderContents" bson:"orderContents" validate:"omitempty,max=255"`
	Misc                       string              `json:"misc" bson:"misc"`
	OrderSumUSD                float64             `json:"orderSumUsd" bson:"orderSumUsd" validate:"min=0"`
	TransportInsurancePermille float64             `json:"transportInsurancePermille" bson:"transportInsurancePermille" validate:"min=0"`
	Discount                   float64             `json:"discount" bson:"discount" validate:"min=0"`
	PreDollarRate              float64             `json:"preDollarRate" bson:"preDollarRate" validate:"min=0"`
	InvoiceFreightCarrierEUR   float64             `json:"invoiceFreightCarrierEur" bson:"invoiceFreightCarrierEur" validate:"min=0"`
	// Status fields
	DeliveryStatus string `json:"deliveryStatus,omitempty" bson:"deliveryStatus,omitempty" validate:"omitempty,oneof=pending ordered shipped arrived partial"`
	PaymentStatus  string `json:"paymentStatus,omitempty" bson:"paymentStatus,omitempty" validate:"omitempty,oneof=unpaid deposit_paid fully_paid overdue"`
	ReceiptStatus  string `json:"receiptStatus,omitempty" bson:"receiptStatus,omitempty" validate:"omitempty,oneof=pending partial received distributed"`
	Tags           []string      `json:"tags,omitempty" bson:"tags,omitempty"`
	Products  []OrderProduct `json:"products" bson:"products"`
	Freights  []OrderFreight `json:"freights,omitempty" bson:"freights,omitempty"`
	Freight   *OrderFreight  `json:"-" bson:"freight,omitempty"` // legacy single-freight field — migrated on read
	LegacyID  int            `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt      time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

type TaskTemplate struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Text      string             `json:"text" bson:"text" validate:"required,min=1,max=500"`
	DaysAfter int                `json:"daysAfter" bson:"daysAfter" validate:"min=0"` // days after trigger event
	Phase     int                `json:"phase" bson:"phase" validate:"min=1,max=2"`   // 1=order placed, 2=shipped
	LegacyID  int                `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type OrderTask struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	OrderID     primitive.ObjectID  `json:"orderId" bson:"orderId" validate:"required"`
	AssigneeID  *primitive.ObjectID `json:"assigneeId,omitempty" bson:"assigneeId,omitempty"`
	Title       string              `json:"title,omitempty" bson:"title,omitempty" validate:"omitempty,max=200"`
	Text        string              `json:"text" bson:"text" validate:"required,min=1,max=500"`
	Status      string              `json:"status,omitempty" bson:"status,omitempty" validate:"omitempty,oneof=open in_progress done"`
	Priority    string              `json:"priority,omitempty" bson:"priority,omitempty" validate:"omitempty,oneof=low medium high"`
	DueDate     *time.Time          `json:"dueDate,omitempty" bson:"dueDate,omitempty"`
	DoneAt      *time.Time          `json:"doneAt,omitempty" bson:"doneAt,omitempty"`
	DoneByID    *primitive.ObjectID `json:"doneById,omitempty" bson:"doneById,omitempty"`
	LegacyID    int                 `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Offers / AngebotsMaster
// ---------------------------------------------------------------------------

type Offer struct {
	ID           primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID     primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	ProductID    *primitive.ObjectID `json:"productId,omitempty" bson:"productId,omitempty"`
	SupplierID   *primitive.ObjectID `json:"supplierId,omitempty" bson:"supplierId,omitempty"`
	IsStockOffer bool                `json:"isStockOffer" bson:"isStockOffer"`
	NameShort    string              `json:"nameShort" bson:"nameShort" validate:"omitempty,max=100"`
	Description  string              `json:"description" bson:"description"`
	PriceUSD     float64             `json:"priceUsd" bson:"priceUsd" validate:"min=0"`
	Quantity     int                 `json:"quantity" bson:"quantity" validate:"min=0"`
	ValidUntil   *time.Time          `json:"validUntil,omitempty" bson:"validUntil,omitempty"`
	Misc         string              `json:"misc" bson:"misc"`
	Tags         []string            `json:"tags,omitempty" bson:"tags,omitempty"`
	LegacyID     int                 `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt    time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Inventory / Stock
// ---------------------------------------------------------------------------

// StockMovement records a goods receipt (Wareneingang), issue (Warenausgang), or adjustment.
// type: receipt | issue | adjustment
type StockMovement struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	ProductID   primitive.ObjectID  `json:"productId" bson:"productId" validate:"required"`
	OrderID     *primitive.ObjectID `json:"orderId,omitempty" bson:"orderId,omitempty"`
	CustomerID  *primitive.ObjectID `json:"customerId,omitempty" bson:"customerId,omitempty"`
	Type        string              `json:"type" bson:"type" validate:"required,oneof=receipt issue adjustment"`
	Quantity    float64             `json:"quantity" bson:"quantity" validate:"required"`
	Unit        string              `json:"unit,omitempty" bson:"unit,omitempty" validate:"omitempty,max=20"`
	Location    string              `json:"location,omitempty" bson:"location,omitempty" validate:"omitempty,max=100"`
	Notes       string              `json:"notes,omitempty" bson:"notes,omitempty" validate:"omitempty,max=500"`
	ProcessedBy primitive.ObjectID  `json:"processedBy" bson:"processedBy"`
	MovedAt     time.Time           `json:"movedAt" bson:"movedAt"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// StockLevel is a cached/computed current stock level per product.
type StockLevel struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	ProductID primitive.ObjectID `json:"productId" bson:"productId" validate:"required"`
	Quantity  float64            `json:"quantity" bson:"quantity"`
	Unit      string             `json:"unit,omitempty" bson:"unit,omitempty" validate:"omitempty,max=20"`
	Location  string             `json:"location,omitempty" bson:"location,omitempty" validate:"omitempty,max=100"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// File attachments
// ---------------------------------------------------------------------------

type ProcurementFile struct {
	ID          primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID  `json:"tenantId" bson:"tenantId" validate:"required"`
	OrderID     *primitive.ObjectID `json:"orderId,omitempty" bson:"orderId,omitempty"`
	ProductID   *primitive.ObjectID `json:"productId,omitempty" bson:"productId,omitempty"`
	Filename    string              `json:"filename" bson:"filename" validate:"required,min=1,max=255"`
	MimeType    string              `json:"mimeType" bson:"mimeType" validate:"omitempty,max=100"`
	SizeBytes   int64               `json:"sizeBytes" bson:"sizeBytes" validate:"min=0"`
	StoragePath string              `json:"storagePath" bson:"storagePath" validate:"required"`
	UploadedBy  primitive.ObjectID  `json:"uploadedBy" bson:"uploadedBy"`
	LegacyID    int                 `json:"legacyId,omitempty" bson:"legacyId,omitempty"`
	CreatedAt   time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt" bson:"updatedAt"`
}

// ---------------------------------------------------------------------------
// Calendar entries (manual)
// ---------------------------------------------------------------------------

type CalendarEntry struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID  primitive.ObjectID `json:"tenantId" bson:"tenantId" validate:"required"`
	Title     string             `json:"title" bson:"title" validate:"required,min=1,max=200"`
	Notes     string             `json:"notes,omitempty" bson:"notes,omitempty" validate:"omitempty,max=1000"`
	Date      time.Time          `json:"date" bson:"date" validate:"required"`
	// type: reminder | milestone | appointment | deadline
	EntryType string `json:"entryType" bson:"entryType" validate:"required,oneof=reminder milestone appointment deadline"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
