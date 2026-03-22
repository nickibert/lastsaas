package xentral

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"lastsaas/internal/db"
	"lastsaas/internal/models"
)

// Engine executes bidirectional sync between Xentral and LastSaaS.
// Direction: Xentral → LastSaaS (inbound) for all four entity types.
// The XentralMapping collection is used as a join table so that repeated
// syncs update existing records rather than inserting duplicates.
type Engine struct {
	db *db.MongoDB
}

func NewEngine(database *db.MongoDB) *Engine {
	return &Engine{db: database}
}

// -------------------------------------------------------------------
// Products (Xentral articles → LastSaaS products)
// -------------------------------------------------------------------

func (e *Engine) SyncProducts(ctx context.Context, tenantID primitive.ObjectID, client *Client) models.XentralSyncLog {
	log := models.XentralSyncLog{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		Entity:    "products",
		Status:    "running",
		StartedAt: time.Now(),
	}
	e.db.XentralSyncLogs().InsertOne(ctx, log) //nolint

	articles, err := client.ListArticles(ctx)
	if err != nil {
		return e.failLog(ctx, log, err.Error())
	}

	for _, a := range articles {
		if err := e.upsertProduct(ctx, tenantID, a); err != nil {
			log.Errors = append(log.Errors, fmt.Sprintf("%s: %v", a.ID, err))
			log.Skipped++
		} else {
			// distinguish create vs update via mapping existence check (already done in upsertProduct)
			log.Updated++
		}
	}

	return e.finishLog(ctx, log)
}

func (e *Engine) upsertProduct(ctx context.Context, tenantID primitive.ObjectID, a XArticle) error {
	if a.ID == "" || a.Name == "" {
		return nil // skip incomplete records
	}

	// Look up existing mapping
	var mapping models.XentralMapping
	err := e.db.XentralMappings().FindOne(ctx, bson.M{
		"tenantId":  tenantID,
		"entity":    "product",
		"xentralId": a.ID,
	}).Decode(&mapping)

	now := time.Now()
	nameShort := truncate(a.Name, 45)
	if nameShort == "" {
		nameShort = truncate(a.ArticleNumber, 45)
	}

	if err == mongo.ErrNoDocuments {
		// Create new product
		product := models.Product{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			NameShort: nameShort,
			NameLong:  truncate(a.Name, 255),
			EAN:       truncate(a.EAN, 32),
			WeightKg:  a.Weight,
			LastEK:    a.PurchasePrice,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := e.db.Products().InsertOne(ctx, product); err != nil {
			return fmt.Errorf("insert product: %w", err)
		}
		_, _ = e.db.XentralMappings().InsertOne(ctx, models.XentralMapping{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			Entity:    "product",
			LocalID:   product.ID,
			XentralID: a.ID,
			XentralNr: a.ArticleNumber,
			CreatedAt: now,
			UpdatedAt: now,
		})
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup mapping: %w", err)
	}

	// Update existing product
	_, _ = e.db.Products().UpdateOne(ctx,
		bson.M{"_id": mapping.LocalID, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"nameShort": nameShort,
			"nameLong":  truncate(a.Name, 255),
			"ean":       truncate(a.EAN, 32),
			"weightKg":  a.Weight,
			"lastEk":    a.PurchasePrice,
			"updatedAt": now,
		}},
	)
	_, _ = e.db.XentralMappings().UpdateOne(ctx,
		bson.M{"_id": mapping.ID},
		bson.M{"$set": bson.M{"xentralNr": a.ArticleNumber, "updatedAt": now}},
	)
	return nil
}

// -------------------------------------------------------------------
// Customers (Xentral customers → LastSaaS customers)
// -------------------------------------------------------------------

func (e *Engine) SyncCustomers(ctx context.Context, tenantID primitive.ObjectID, client *Client) models.XentralSyncLog {
	log := models.XentralSyncLog{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		Entity:    "customers",
		Status:    "running",
		StartedAt: time.Now(),
	}
	e.db.XentralSyncLogs().InsertOne(ctx, log) //nolint

	customers, err := client.ListCustomers(ctx)
	if err != nil {
		return e.failLog(ctx, log, err.Error())
	}

	for _, c := range customers {
		if err := e.upsertCustomer(ctx, tenantID, c); err != nil {
			log.Errors = append(log.Errors, fmt.Sprintf("%s: %v", c.ID, err))
			log.Skipped++
		} else {
			log.Updated++
		}
	}

	return e.finishLog(ctx, log)
}

func (e *Engine) upsertCustomer(ctx context.Context, tenantID primitive.ObjectID, xc XCustomer) error {
	if xc.ID == "" {
		return nil
	}
	company := xc.Company
	if company == "" {
		company = strings.TrimSpace(xc.FirstName + " " + xc.LastName)
	}
	if company == "" {
		return nil
	}

	var mapping models.XentralMapping
	err := e.db.XentralMappings().FindOne(ctx, bson.M{
		"tenantId":  tenantID,
		"entity":    "customer",
		"xentralId": xc.ID,
	}).Decode(&mapping)

	now := time.Now()
	addr := models.CustomerAddress{
		Street:  xc.Address.Street,
		City:    xc.Address.City,
		Zip:     xc.Address.Postcode,
		Country: xc.Address.Country,
	}

	if err == mongo.ErrNoDocuments {
		customer := models.Customer{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			Company:   truncate(company, 200),
			Firstname: truncate(xc.FirstName, 50),
			Lastname:  truncate(xc.LastName, 50),
			Email:     truncate(xc.Email, 120),
			Phone:     truncate(xc.Phone, 50),
			Address:   addr,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := e.db.Customers().InsertOne(ctx, customer); err != nil {
			return fmt.Errorf("insert customer: %w", err)
		}
		_, _ = e.db.XentralMappings().InsertOne(ctx, models.XentralMapping{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			Entity:    "customer",
			LocalID:   customer.ID,
			XentralID: xc.ID,
			XentralNr: xc.CustomerNumber,
			CreatedAt: now,
			UpdatedAt: now,
		})
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup mapping: %w", err)
	}

	_, _ = e.db.Customers().UpdateOne(ctx,
		bson.M{"_id": mapping.LocalID, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"company":   truncate(company, 200),
			"firstname": truncate(xc.FirstName, 50),
			"lastname":  truncate(xc.LastName, 50),
			"email":     truncate(xc.Email, 120),
			"phone":     truncate(xc.Phone, 50),
			"address":   addr,
			"updatedAt": now,
		}},
	)
	return nil
}

// -------------------------------------------------------------------
// Suppliers (Xentral suppliers → LastSaaS suppliers)
// -------------------------------------------------------------------

func (e *Engine) SyncSuppliers(ctx context.Context, tenantID primitive.ObjectID, client *Client) models.XentralSyncLog {
	log := models.XentralSyncLog{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		Entity:    "suppliers",
		Status:    "running",
		StartedAt: time.Now(),
	}
	e.db.XentralSyncLogs().InsertOne(ctx, log) //nolint

	suppliers, err := client.ListSuppliers(ctx)
	if err != nil {
		return e.failLog(ctx, log, err.Error())
	}

	for _, s := range suppliers {
		if err := e.upsertSupplier(ctx, tenantID, s); err != nil {
			log.Errors = append(log.Errors, fmt.Sprintf("%s: %v", s.ID, err))
			log.Skipped++
		} else {
			log.Updated++
		}
	}

	return e.finishLog(ctx, log)
}

func (e *Engine) upsertSupplier(ctx context.Context, tenantID primitive.ObjectID, xs XSupplier) error {
	if xs.ID == "" {
		return nil
	}
	company := xs.Company
	if company == "" {
		company = strings.TrimSpace(xs.FirstName + " " + xs.LastName)
	}
	if company == "" {
		return nil
	}

	var mapping models.XentralMapping
	err := e.db.XentralMappings().FindOne(ctx, bson.M{
		"tenantId":  tenantID,
		"entity":    "supplier",
		"xentralId": xs.ID,
	}).Decode(&mapping)

	now := time.Now()

	if err == mongo.ErrNoDocuments {
		supplier := models.Supplier{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			Company:   truncate(company, 200),
			Firstname: truncate(xs.FirstName, 50),
			Lastname:  truncate(xs.LastName, 50),
			Email:     truncate(xs.Email, 120),
			Origin:    truncate(xs.Origin, 50),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if _, err := e.db.Suppliers().InsertOne(ctx, supplier); err != nil {
			return fmt.Errorf("insert supplier: %w", err)
		}
		_, _ = e.db.XentralMappings().InsertOne(ctx, models.XentralMapping{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			Entity:    "supplier",
			LocalID:   supplier.ID,
			XentralID: xs.ID,
			XentralNr: xs.SupplierNumber,
			CreatedAt: now,
			UpdatedAt: now,
		})
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup mapping: %w", err)
	}

	_, _ = e.db.Suppliers().UpdateOne(ctx,
		bson.M{"_id": mapping.LocalID, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"company":   truncate(company, 200),
			"firstname": truncate(xs.FirstName, 50),
			"lastname":  truncate(xs.LastName, 50),
			"email":     truncate(xs.Email, 120),
			"origin":    truncate(xs.Origin, 50),
			"updatedAt": now,
		}},
	)
	return nil
}

// -------------------------------------------------------------------
// Orders (Xentral sales-orders → LastSaaS orders, inbound)
// -------------------------------------------------------------------

func (e *Engine) SyncOrders(ctx context.Context, tenantID primitive.ObjectID, client *Client) models.XentralSyncLog {
	log := models.XentralSyncLog{
		ID:        primitive.NewObjectID(),
		TenantID:  tenantID,
		Entity:    "orders",
		Status:    "running",
		StartedAt: time.Now(),
	}
	e.db.XentralSyncLogs().InsertOne(ctx, log) //nolint

	orders, err := client.ListSalesOrders(ctx)
	if err != nil {
		return e.failLog(ctx, log, err.Error())
	}

	for _, o := range orders {
		if err := e.upsertOrder(ctx, tenantID, o); err != nil {
			log.Errors = append(log.Errors, fmt.Sprintf("%s: %v", o.ID, err))
			log.Skipped++
		} else {
			log.Updated++
		}
	}

	return e.finishLog(ctx, log)
}

func (e *Engine) upsertOrder(ctx context.Context, tenantID primitive.ObjectID, xo XSalesOrder) error {
	if xo.ID == "" {
		return nil
	}

	var mapping models.XentralMapping
	err := e.db.XentralMappings().FindOne(ctx, bson.M{
		"tenantId":  tenantID,
		"entity":    "order",
		"xentralId": xo.ID,
	}).Decode(&mapping)

	now := time.Now()

	// Resolve order date (try orderDate, fall back to documentDate)
	orderDate := now
	if d := xo.ResolvedOrderDate(); d != "" {
		if t, parseErr := time.Parse("2006-01-02", d); parseErr == nil {
			orderDate = t
		}
	}

	// Build order products from positions
	var orderProducts []models.OrderProduct
	for _, pos := range xo.Positions {
		orderProducts = append(orderProducts, models.OrderProduct{
			Quantity:      int(pos.Quantity),
			UnitPriceUSD:  pos.UnitPrice,
			TotalPriceUSD: pos.UnitPrice * pos.Quantity,
		})
	}

	if err == mongo.ErrNoDocuments {
		order := models.Order{
			ID:          primitive.NewObjectID(),
			TenantID:    tenantID,
			OrderNumber: xo.ResolvedOrderNumber(),
			OrderDate:   orderDate,
			OrderSumUSD: xo.TotalNet,
			Products:    orderProducts,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if _, insertErr := e.db.Orders().InsertOne(ctx, order); insertErr != nil {
			return fmt.Errorf("insert order: %w", insertErr)
		}
		_, _ = e.db.XentralMappings().InsertOne(ctx, models.XentralMapping{
			ID:        primitive.NewObjectID(),
			TenantID:  tenantID,
			Entity:    "order",
			LocalID:   order.ID,
			XentralID: xo.ID,
			XentralNr: xo.ResolvedOrderNumber(),
			CreatedAt: now,
			UpdatedAt: now,
		})
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup mapping: %w", err)
	}

	_, _ = e.db.Orders().UpdateOne(ctx,
		bson.M{"_id": mapping.LocalID, "tenantId": tenantID},
		bson.M{"$set": bson.M{
			"orderSumUsd": xo.TotalNet,
			"products":    orderProducts,
			"updatedAt":   now,
		}},
	)
	return nil
}

// -------------------------------------------------------------------
// Scheduler
// -------------------------------------------------------------------

// RunScheduler starts a background goroutine that triggers auto-sync
// for all enabled tenants according to their configured SyncIntervalH.
// Call once from main; returns a stop function.
func RunScheduler(ctx context.Context, database *db.MongoDB) {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runScheduledSyncs(ctx, database)
			}
		}
	}()
}

func runScheduledSyncs(ctx context.Context, database *db.MongoDB) {
	cursor, err := database.XentralConfigs().Find(ctx, bson.M{
		"enabled":       true,
		"syncIntervalH": bson.M{"$gt": 0},
	}, options.Find().SetProjection(bson.M{
		"tenantId": 1, "baseUrl": 1, "apiToken": 1,
		"syncIntervalH": 1,
		"syncProducts": 1, "syncCustomers": 1, "syncSuppliers": 1, "syncOrders": 1,
		"lastSyncProducts": 1, "lastSyncCustomers": 1, "lastSyncSuppliers": 1, "lastSyncOrders": 1,
	}))
	if err != nil {
		return
	}
	var configs []models.XentralConfig
	cursor.All(ctx, &configs) //nolint

	engine := NewEngine(database)
	now := time.Now()

	for _, cfg := range configs {
		client := NewClient(cfg.BaseURL, cfg.APIToken)
		interval := time.Duration(cfg.SyncIntervalH) * time.Hour

		if cfg.SyncProducts && isDue(cfg.LastSyncProducts, now, interval) {
			engine.SyncProducts(ctx, cfg.TenantID, client) //nolint
			t := now
			database.XentralConfigs().UpdateOne(ctx, //nolint
				bson.M{"tenantId": cfg.TenantID},
				bson.M{"$set": bson.M{"lastSyncProducts": t}},
			)
		}
		if cfg.SyncCustomers && isDue(cfg.LastSyncCustomers, now, interval) {
			engine.SyncCustomers(ctx, cfg.TenantID, client) //nolint
			t := now
			database.XentralConfigs().UpdateOne(ctx, //nolint
				bson.M{"tenantId": cfg.TenantID},
				bson.M{"$set": bson.M{"lastSyncCustomers": t}},
			)
		}
		if cfg.SyncSuppliers && isDue(cfg.LastSyncSuppliers, now, interval) {
			engine.SyncSuppliers(ctx, cfg.TenantID, client) //nolint
			t := now
			database.XentralConfigs().UpdateOne(ctx, //nolint
				bson.M{"tenantId": cfg.TenantID},
				bson.M{"$set": bson.M{"lastSyncSuppliers": t}},
			)
		}
		if cfg.SyncOrders && isDue(cfg.LastSyncOrders, now, interval) {
			engine.SyncOrders(ctx, cfg.TenantID, client) //nolint
			t := now
			database.XentralConfigs().UpdateOne(ctx, //nolint
				bson.M{"tenantId": cfg.TenantID},
				bson.M{"$set": bson.M{"lastSyncOrders": t}},
			)
		}
	}
}

func isDue(last *time.Time, now time.Time, interval time.Duration) bool {
	if last == nil {
		return true
	}
	return now.Sub(*last) >= interval
}

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

func (e *Engine) failLog(ctx context.Context, log models.XentralSyncLog, errMsg string) models.XentralSyncLog {
	now := time.Now()
	log.Status = "error"
	log.Errors = []string{errMsg}
	log.FinishedAt = &now
	e.db.XentralSyncLogs().UpdateOne(ctx, //nolint
		bson.M{"_id": log.ID},
		bson.M{"$set": bson.M{"status": log.Status, "errors": log.Errors, "finishedAt": log.FinishedAt}},
	)
	return log
}

func (e *Engine) finishLog(ctx context.Context, log models.XentralSyncLog) models.XentralSyncLog {
	now := time.Now()
	log.Status = "success"
	if len(log.Errors) > 0 {
		log.Status = "partial"
	}
	log.FinishedAt = &now
	e.db.XentralSyncLogs().UpdateOne(ctx, //nolint
		bson.M{"_id": log.ID},
		bson.M{"$set": bson.M{
			"status":     log.Status,
			"created":    log.Created,
			"updated":    log.Updated,
			"skipped":    log.Skipped,
			"errors":     log.Errors,
			"finishedAt": log.FinishedAt,
		}},
	)
	return log
}

// Truncate is the exported version used by the handlers package.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func truncate(s string, max int) string { return Truncate(s, max) }
