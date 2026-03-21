// migrate-easyonedata imports the legacy EasyOne MySQL dump into MongoDB.
//
// Usage:
//
//	go run ./cmd/migrate-easyonedata \
//	  --sql /path/to/easyone.sql \
//	  --mongo mongodb://localhost:27017 \
//	  --db lastsaas \
//	  --tenant <tenantId>
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---------------------------------------------------------------------------
// CLI flags
// ---------------------------------------------------------------------------

var (
	flagSQL      = flag.String("sql", "", "Path to easyone.sql dump (required)")
	flagMongo    = flag.String("mongo", "mongodb://localhost:27017", "MongoDB URI")
	flagDB       = flag.String("db", "lastsaas", "MongoDB database name")
	flagTenantID = flag.String("tenant", "", "MongoDB ObjectID of target tenant (required)")
	flagDryRun   = flag.Bool("dry-run", false, "Parse only, do not write to MongoDB")
)

// ---------------------------------------------------------------------------
// Minimal parsed representations matching the Go models
// ---------------------------------------------------------------------------

type Row = map[string]string

// ---------------------------------------------------------------------------
// SQL parsing
// ---------------------------------------------------------------------------

var insertRe = regexp.MustCompile(`^INSERT INTO ` + "`" + `(\w+)` + "`" + ` VALUES (.+);$`)

// parseSQLValue converts a SQL literal (e.g. 'foo', NULL, 42) to string.
func parseSQLValue(s string) string {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "NULL") {
		return ""
	}
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
		s = strings.ReplaceAll(s, "''", "'")
		s = strings.ReplaceAll(s, `\'`, "'")
		s = strings.ReplaceAll(s, `\\`, `\`)
		return s
	}
	return s
}

// splitValues splits a VALUES tuple string like ('a','b',1,NULL) into a slice of raw tokens.
func splitValues(s string) []string {
	// Strip outer parentheses
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '(' && s[len(s)-1:] == ")" {
		s = s[1 : len(s)-1]
	}
	var result []string
	var cur strings.Builder
	inStr := false
	escaped := false
	for _, ch := range s {
		if escaped {
			cur.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' && inStr {
			cur.WriteRune(ch)
			escaped = true
			continue
		}
		if ch == '\'' {
			inStr = !inStr
			cur.WriteRune(ch)
			continue
		}
		if ch == ',' && !inStr {
			result = append(result, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(ch)
	}
	if cur.Len() > 0 {
		result = append(result, cur.String())
	}
	return result
}

// tableColumns returns the columns for known tables.
func tableColumns(table string) []string {
	cols := map[string][]string{
		"supplier": {"supplier_id", "company", "firstname", "lastname", "email", "reg_date", "icq", "msn", "skype", "misc", "origin"},
		"goods_groups": {"goods_group_id", "name", "short"},
		"products": {"products_id", "name_short", "own_name_short", "name_long", "description", "reg_date", "misc", "preview_pic", "description_pic",
			"checked", "wtn", "ean", "userdef01", "userdef02", "userdef03", "userdef04", "userdef05",
			"userdef06", "userdef07", "userdef08", "userdef09", "userdef10",
			"width", "height", "length", "weight", "vpe", "goods_group_id", "supplier_code_id", "aco", "last_ek", "last_ek_date", "virtual"},
		"freight_carrier": {"freight_carrier_id", "name", "description"},
		"harbour": {"harbour_id", "name", "description"},
		"container": {"container_id", "name", "description", "volume", "height_m", "length_m", "width_m"},
		"countries": {"country_id", "name"},
		"orders": {"orders_id", "supplier_id", "order_date", "misc", "pics", "misc_files", "order_sum_usd",
			"transport_insurance", "invoice_freight_carrier", "invoice_freight_carrier_eur", "discount", "order_number", "order_contents", "pre_dollar_rate"},
		"orders_products": {"orders_id", "products_id", "quantity", "unit_price_usd", "total_price_usd",
			"length_mm", "width_mm", "height_mm", "volume", "weight_kg", "credited", "inventory_checked"},
		"orders_payments": {"payment_id", "orders_id", "nr", "payment_amount", "payment_date", "payment_dollar_rate", "payment_fees"},
		"orders_freight": {"orders_freight_id", "container_id", "harbour_id_from", "harbour_id_to", "orders_id",
			"freight_carrier_id", "proforma_invoice", "commercial_invoice", "estimated_arrival", "shipping_date",
			"packing_list", "avis_shipper_date", "doc_of_origin", "doo_checked", "doo_signed", "doo_shipped",
			"arrival", "container_nr", "freighter_invoice", "freightage", "pre_freightage",
			"sea_freight_usd", "emergency_bunker_surcharge_usd", "peak_season_surcharge_usd",
			"suez_canal_addon_usd", "danger_pay_usd", "dollar_rate",
			"thc_eur", "isps_eur", "bl_doc_fee_eur", "follow_up_fees_oldb_eur",
			"customs_clearance_eur", "customs_eur", "customs_percent", "ztn"},
		"products_supplier": {"products_id", "supplier_id"},
		"tasks":      {"task_id", "date_id", "plus", "text"},
		"tasks_done": {"id", "task_id", "orders_id", "user_id"},
		"tasks_todo": {"task_id", "user_id", "orders_id"},
	}
	return cols[table]
}

// parseDump reads the SQL file and returns rows grouped by table name.
func parseDump(path string) (map[string][]Row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string][]Row)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := insertRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		table := m[1]
		cols := tableColumns(table)
		if cols == nil {
			continue // skip uninteresting tables
		}
		// m[2] may be a single tuple or multiple tuples separated by '),('
		tupleStr := m[2]
		// Split multiple tuples: ),( is the separator
		tuples := splitTuples(tupleStr)
		for _, tuple := range tuples {
			vals := splitValues(tuple)
			if len(vals) != len(cols) {
				continue
			}
			row := make(Row, len(cols))
			for i, col := range cols {
				row[col] = parseSQLValue(vals[i])
			}
			result[table] = append(result[table], row)
		}
	}
	return result, scanner.Err()
}

// splitTuples splits "(...),(...)" into individual tuple strings.
func splitTuples(s string) []string {
	var tuples []string
	depth := 0
	inStr := false
	escaped := false
	start := 0
	for i, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inStr {
			escaped = true
			continue
		}
		if ch == '\'' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		if ch == '(' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				tuples = append(tuples, s[start:i+1])
			}
		}
	}
	return tuples
}

// ---------------------------------------------------------------------------
// ID mapping: legacy int → MongoDB ObjectID
// ---------------------------------------------------------------------------

type IDMap struct {
	tenantID primitive.ObjectID
	m        map[string]primitive.ObjectID
}

func newIDMap(tenantID primitive.ObjectID) *IDMap {
	return &IDMap{tenantID: tenantID, m: make(map[string]primitive.ObjectID)}
}

// deterministicID returns a stable ObjectID derived from tenantID+table+legacyID
// so that re-running the migration produces the same IDs and upserts correctly.
func deterministicID(tenantID primitive.ObjectID, table, legacyID string) primitive.ObjectID {
	h := sha256.Sum256([]byte(tenantID.Hex() + ":" + table + ":" + legacyID))
	var oid primitive.ObjectID
	copy(oid[:], h[:12])
	return oid
}

func (im *IDMap) get(table, legacyID string) primitive.ObjectID {
	key := table + ":" + legacyID
	if id, ok := im.m[key]; ok {
		return id
	}
	id := deterministicID(im.tenantID, table, legacyID)
	im.m[key] = id
	return id
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// coalesce returns the first non-empty string from the arguments.
func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// truncate cuts s to at most n runes (MongoDB schema maxLength).
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func parseBool(s string) bool {
	return s == "1"
}

func parseDate(s string) *time.Time {
	if s == "" || s == "0000-00-00" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func nowUTC() time.Time { return time.Now().UTC() }

// ---------------------------------------------------------------------------
// Import logic
// ---------------------------------------------------------------------------

func importData(ctx context.Context, db *mongo.Database, tenantID primitive.ObjectID, rows map[string][]Row, dryRun bool) {
	idMap := newIDMap(tenantID)
	now := nowUTC()

	type doc = map[string]any

	insertAll := func(collection string, docs []doc) {
		if len(docs) == 0 {
			return
		}
		log.Printf("[%s] %d documents", collection, len(docs))
		if dryRun {
			sample, _ := json.Marshal(docs[0])
			log.Printf("  sample: %s", sample)
			return
		}
		const batchSize = 500
		total := 0
		for start := 0; start < len(docs); start += batchSize {
			end := start + batchSize
			if end > len(docs) {
				end = len(docs)
			}
			batch := docs[start:end]
			models := make([]mongo.WriteModel, len(batch))
			for i, d := range batch {
				models[i] = mongo.NewReplaceOneModel().
					SetFilter(bson.M{"_id": d["_id"]}).
					SetReplacement(d).
					SetUpsert(true)
			}
			res, err := db.Collection(collection).BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
			if err != nil {
				log.Printf("  batch %d-%d ERROR: %v", start, end, err)
			} else {
				total += int(res.UpsertedCount + res.ModifiedCount + res.InsertedCount)
			}
		}
		log.Printf("  upserted %d total", total)
	}

	// --- suppliers ---
	{
		var docs []doc
		for _, r := range rows["supplier"] {
			docs = append(docs, doc{
				"_id":      idMap.get("supplier", r["supplier_id"]),
				"tenantId": tenantID,
				"company":  coalesce(r["company"], r["lastname"], r["firstname"], "(unbekannt)"),
				"firstname": r["firstname"],
				"lastname":  r["lastname"],
				"email":     r["email"],
				"skype":     r["skype"],
				"origin":    r["origin"],
				"misc":      r["misc"],
				"legacyId":  parseInt(r["supplier_id"]),
				"createdAt": now,
				"updatedAt": now,
			})
		}
		insertAll("suppliers", docs)
	}

	// --- goods_groups ---
	{
		var docs []doc
		for _, r := range rows["goods_groups"] {
			docs = append(docs, doc{
				"_id":       idMap.get("goods_group", r["goods_group_id"]),
				"tenantId":  tenantID,
				"name":      r["name"],
				"short":     truncate(r["short"], 100),
				"legacyId":  parseInt(r["goods_group_id"]),
				"createdAt": now,
				"updatedAt": now,
			})
		}
		insertAll("goods_groups", docs)
	}

	// --- freight_carrier ---
	{
		var docs []doc
		for _, r := range rows["freight_carrier"] {
			docs = append(docs, doc{
				"_id":         idMap.get("freight_carrier", r["freight_carrier_id"]),
				"tenantId":    tenantID,
				"name":        r["name"],
				"description": r["description"],
				"legacyId":    parseInt(r["freight_carrier_id"]),
				"createdAt":   now,
				"updatedAt":   now,
			})
		}
		insertAll("freight_carriers", docs)
	}

	// --- harbour ---
	{
		var docs []doc
		for _, r := range rows["harbour"] {
			docs = append(docs, doc{
				"_id":         idMap.get("harbour", r["harbour_id"]),
				"tenantId":    tenantID,
				"name":        r["name"],
				"description": r["description"],
				"legacyId":    parseInt(r["harbour_id"]),
				"createdAt":   now,
				"updatedAt":   now,
			})
		}
		insertAll("harbours", docs)
	}

	// --- container ---
	{
		var docs []doc
		for _, r := range rows["container"] {
			docs = append(docs, doc{
				"_id":         idMap.get("container", r["container_id"]),
				"tenantId":    tenantID,
				"name":        r["name"],
				"description": r["description"],
				"volumeM3":    parseFloat(r["volume"]),
				"heightM":     parseFloat(r["height_m"]),
				"lengthM":     parseFloat(r["length_m"]),
				"widthM":      parseFloat(r["width_m"]),
				"legacyId":    parseInt(r["container_id"]),
				"createdAt":   now,
				"updatedAt":   now,
			})
		}
		insertAll("containers", docs)
	}

	// --- countries ---
	{
		var docs []doc
		for _, r := range rows["countries"] {
			docs = append(docs, doc{
				"_id":       idMap.get("country", r["country_id"]),
				"tenantId":  tenantID,
				"name":      r["name"],
				"legacyId":  parseInt(r["country_id"]),
				"createdAt": now,
				"updatedAt": now,
			})
		}
		insertAll("countries", docs)
	}

	// --- products ---
	{
		var docs []doc
		for _, r := range rows["products"] {
			d := doc{
				"_id":          idMap.get("product", r["products_id"]),
				"tenantId":     tenantID,
				"nameShort":    truncate(coalesce(r["name_short"], r["own_name_short"], r["name_long"], "?"), 45),
				"ownNameShort": truncate(r["own_name_short"], 45),
				"nameLong":     r["name_long"],
				"description":  r["description"],
				"misc":         r["misc"],
				"wtn":          truncate(r["wtn"], 30),
				"ean":          truncate(r["ean"], 32),
				"aco":          r["aco"],
				"widthMm":      parseInt(r["width"]),
				"heightMm":     parseInt(r["height"]),
				"lengthMm":     parseInt(r["length"]),
				"weightKg":     parseFloat(r["weight"]),
				"vpe":          parseInt(r["vpe"]),
				"lastEk":       parseFloat(r["last_ek"]),
				"checked":      parseBool(r["checked"]),
				"virtual":      parseBool(r["virtual"]),
				"userDef01":    r["userdef01"],
				"userDef02":    r["userdef02"],
				"userDef03":    r["userdef03"],
				"userDef04":    r["userdef04"],
				"userDef05":    r["userdef05"],
				"userDef06":    r["userdef06"],
				"userDef07":    r["userdef07"],
				"userDef08":    r["userdef08"],
				"userDef09":    r["userdef09"],
				"userDef10":    r["userdef10"],
				"legacyId":     parseInt(r["products_id"]),
				"createdAt":    now,
				"updatedAt":    now,
			}
			if r["goods_group_id"] != "" && r["goods_group_id"] != "0" {
				d["goodsGroupId"] = idMap.get("goods_group", r["goods_group_id"])
			}
			if dt := parseDate(r["last_ek_date"]); dt != nil {
				d["lastEkDate"] = dt
			}
			docs = append(docs, d)
		}
		insertAll("products", docs)
	}

	// Build products_supplier index: products_id → []supplierOID
	suppliersByProduct := make(map[string][]primitive.ObjectID)
	for _, r := range rows["products_supplier"] {
		pid := r["products_id"]
		sid := r["supplier_id"]
		if pid == "" || sid == "" || sid == "0" {
			continue
		}
		oid := idMap.get("supplier", sid)
		suppliersByProduct[pid] = append(suppliersByProduct[pid], oid)
	}
	// Second pass: update products with supplierIds (and primary supplierId if not set)
	if len(suppliersByProduct) > 0 && !dryRun {
		log.Printf("[products] patching %d products with supplierIds", len(suppliersByProduct))
		patched := 0
		for pid, sids := range suppliersByProduct {
			productOID := idMap.get("product", pid)
			update := bson.M{"supplierIds": sids}
			// Set primary supplierId to first entry if not already set
			update["supplierId"] = sids[0]
			_, err := db.Collection("products").UpdateOne(ctx,
				bson.M{"_id": productOID, "tenantId": tenantID},
				bson.M{"$set": update},
			)
			if err == nil {
				patched++
			}
		}
		log.Printf("[products] patched supplierIds for %d products", patched)
	}

	// Build orders_freight index: orders_id → freight doc
	freightByOrder := make(map[string]doc)
	for _, r := range rows["orders_freight"] {
		oid := r["orders_id"]
		if oid == "" {
			continue
		}
		f := doc{
			"containerId":      idMap.get("container", r["container_id"]),
			"harbourIdFrom":    idMap.get("harbour", r["harbour_id_from"]),
			"harbourIdTo":      idMap.get("harbour", r["harbour_id_to"]),
			"freightCarrierId": idMap.get("freight_carrier", r["freight_carrier_id"]),
			"containerNr":      r["container_nr"],
			// Dates
			"docOfOriginChecked":  parseBool(r["doo_checked"]),
			"docOfOriginSigned":   parseBool(r["doo_signed"]),
			"docOfOriginShipped":  parseBool(r["doo_shipped"]),
			// Sea freight (USD)
			"seaFreightUsd":               parseFloat(r["sea_freight_usd"]),
			"emergencyBunkerSurchargeUsd": parseFloat(r["emergency_bunker_surcharge_usd"]),
			"peakSeasonSurchargeUsd":      parseFloat(r["peak_season_surcharge_usd"]),
			"suezCanalAddonUsd":           parseFloat(r["suez_canal_addon_usd"]),
			"dangerPayUsd":                parseFloat(r["danger_pay_usd"]),
			"dollarRate":                  parseFloat(r["dollar_rate"]),
			// Domestic / port costs (EUR)
			"freightageEur":       parseFloat(r["freightage"]),
			"preFreightageEur":    parseFloat(r["pre_freightage"]),
			"thcEur":              parseFloat(r["thc_eur"]),
			"ispsEur":             parseFloat(r["isps_eur"]),
			"blDocFeeEur":         parseFloat(r["bl_doc_fee_eur"]),
			"followUpFeesEur":     parseFloat(r["follow_up_fees_oldb_eur"]),
			"customsClearanceEur": parseFloat(r["customs_clearance_eur"]),
			"customsEur":          parseFloat(r["customs_eur"]),
			"customsPercent":      parseFloat(r["customs_percent"]),
			"ztn":                 r["ztn"],
		}
		if dt := parseDate(r["shipping_date"]); dt != nil {
			f["shippingDate"] = dt
		}
		if dt := parseDate(r["estimated_arrival"]); dt != nil {
			f["estimatedArrival"] = dt
		}
		if dt := parseDate(r["arrival"]); dt != nil {
			f["arrival"] = dt
		}
		if dt := parseDate(r["avis_shipper_date"]); dt != nil {
			f["avisShipperDate"] = dt
		}
		if dt := parseDate(r["doc_of_origin"]); dt != nil {
			f["docOfOrigin"] = dt
		}
		// Last freight entry per order wins (orders can have multiple freight legs)
		freightByOrder[oid] = f
	}
	log.Printf("[orders_freight] mapped %d freight records to orders", len(freightByOrder))

	// Build orders_products index: orders_id → []orderProduct
	opByOrder := make(map[string][]doc)
	for _, r := range rows["orders_products"] {
		oid := r["orders_id"]
		opByOrder[oid] = append(opByOrder[oid], doc{
			"productId":        idMap.get("product", r["products_id"]),
			"quantity":         parseInt(r["quantity"]),
			"unitPriceUsd":     parseFloat(r["unit_price_usd"]),
			"totalPriceUsd":    parseFloat(r["total_price_usd"]),
			"lengthMm":         parseInt(r["length_mm"]),
			"widthMm":          parseInt(r["width_mm"]),
			"heightMm":         parseInt(r["height_mm"]),
			"volumeM3":         parseFloat(r["volume"]),
			"weightKg":         parseFloat(r["weight_kg"]),
			"credited":         parseBool(r["credited"]),
			"inventoryChecked": parseBool(r["inventory_checked"]),
		})
	}

	// --- orders ---
	{
		var docs []doc
		for _, r := range rows["orders"] {
			d := doc{
				"_id":                      idMap.get("order", r["orders_id"]),
				"tenantId":                 tenantID,
				"orderNumber":              r["order_number"],
				"orderContents":            r["order_contents"],
				"misc":                     r["misc"],
				"orderSumUsd":              parseFloat(r["order_sum_usd"]),
				"transportInsurancePermille": parseFloat(r["transport_insurance"]),
				"discount":                 parseFloat(r["discount"]),
				"preDollarRate":            parseFloat(r["pre_dollar_rate"]),
				"invoiceFreightCarrierEur": parseFloat(r["invoice_freight_carrier_eur"]),
				"products":                 opByOrder[r["orders_id"]],
				"legacyId":                 parseInt(r["orders_id"]),
				"createdAt":                now,
				"updatedAt":                now,
			}
			if dt := parseDate(r["order_date"]); dt != nil {
				d["orderDate"] = *dt
			} else {
				d["orderDate"] = now
			}
			if r["supplier_id"] != "" && r["supplier_id"] != "0" {
				d["supplierId"] = idMap.get("supplier", r["supplier_id"])
			}
			if f, ok := freightByOrder[r["orders_id"]]; ok {
				d["freight"] = f
			}
			docs = append(docs, d)
		}
		insertAll("orders", docs)
	}

	// --- order_payments ---
	{
		var docs []doc
		for _, r := range rows["orders_payments"] {
			d := doc{
				"_id":               idMap.get("payment", r["payment_id"]),
				"tenantId":          tenantID,
				"orderId":           idMap.get("order", r["orders_id"]),
				"nr":                parseInt(r["nr"]),
				"paymentAmountEur":  parseFloat(r["payment_amount"]),
				"paymentDollarRate": parseFloat(r["payment_dollar_rate"]),
				"paymentFees":       parseFloat(r["payment_fees"]),
				"legacyId":          parseInt(r["payment_id"]),
				"createdAt":         now,
				"updatedAt":         now,
			}
			if dt := parseDate(r["payment_date"]); dt != nil {
				d["paymentDate"] = dt
			}
			docs = append(docs, d)
		}
		insertAll("order_payments", docs)
	}

	// --- task_templates ---
	// tasks.date_id: 1=order_date (phase 1), 2=shipping_date (phase 2), 3=arrival (phase 2)
	{
		var docs []doc
		for _, r := range rows["tasks"] {
			phase := 1
			if r["date_id"] == "2" || r["date_id"] == "3" {
				phase = 2
			}
			docs = append(docs, doc{
				"_id":       idMap.get("task_template", r["task_id"]),
				"tenantId":  tenantID,
				"text":      truncate(r["text"], 500),
				"daysAfter": parseInt(r["plus"]),
				"phase":     phase,
				"legacyId":  parseInt(r["task_id"]),
				"createdAt": now,
				"updatedAt": now,
			})
		}
		insertAll("task_templates", docs)
	}

	// --- order_tasks (from tasks_done and tasks_todo) ---
	// Build task text lookup: task_id -> text
	taskText := make(map[string]string)
	for _, r := range rows["tasks"] {
		taskText[r["task_id"]] = r["text"]
	}
	// Deduplicate tasks_done: (task_id, orders_id) -> first record wins
	type taskOrderKey struct{ taskID, ordersID string }
	doneSet := make(map[taskOrderKey]bool)
	{
		var docs []doc
		for _, r := range rows["tasks_done"] {
			key := taskOrderKey{r["task_id"], r["orders_id"]}
			if doneSet[key] {
				continue // already added this (task, order) combination
			}
			doneSet[key] = true
			text := taskText[r["task_id"]]
			if text == "" {
				continue
			}
			doneAt := now
			docs = append(docs, doc{
				"_id":       idMap.get("order_task_done", r["task_id"]+":"+r["orders_id"]),
				"tenantId":  tenantID,
				"orderId":   idMap.get("order", r["orders_id"]),
				"text":      truncate(text, 500),
				"doneAt":    &doneAt,
				"legacyId":  parseInt(r["task_id"]),
				"createdAt": now,
				"updatedAt": now,
			})
		}
		insertAll("order_tasks", docs)
	}
	// tasks_todo: pending tasks not already present as done
	{
		var docs []doc
		for _, r := range rows["tasks_todo"] {
			key := taskOrderKey{r["task_id"], r["orders_id"]}
			if doneSet[key] {
				continue // already migrated as completed
			}
			text := taskText[r["task_id"]]
			if text == "" {
				continue
			}
			docs = append(docs, doc{
				"_id":       idMap.get("order_task_todo", r["task_id"]+":"+r["orders_id"]),
				"tenantId":  tenantID,
				"orderId":   idMap.get("order", r["orders_id"]),
				"text":      truncate(text, 500),
				"legacyId":  parseInt(r["task_id"]),
				"createdAt": now,
				"updatedAt": now,
			})
		}
		insertAll("order_tasks", docs)
	}
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	flag.Parse()

	if *flagSQL == "" {
		log.Fatal("--sql is required")
	}
	if *flagTenantID == "" {
		log.Fatal("--tenant is required")
	}

	tenantOID, err := primitive.ObjectIDFromHex(*flagTenantID)
	if err != nil {
		log.Fatalf("invalid --tenant: %v", err)
	}

	log.Println("Parsing SQL dump:", *flagSQL)
	rows, err := parseDump(*flagSQL)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}

	for table, rs := range rows {
		log.Printf("  %-25s %d rows", table, len(rs))
	}

	if *flagDryRun {
		log.Println("Dry-run mode — skipping MongoDB write")
		importData(context.Background(), nil, tenantOID, rows, true)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(*flagMongo))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(ctx) //nolint

	db := client.Database(*flagDB)
	log.Printf("Writing to %s/%s as tenant %s", *flagMongo, *flagDB, *flagTenantID)
	importData(ctx, db, tenantOID, rows, false)
	log.Println("Done.")
}
