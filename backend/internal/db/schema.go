package db

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// CollectionSchema pairs a collection name with its JSON Schema validator.
type CollectionSchema struct {
	Collection string
	Schema     bson.M
}

// AllSchemas returns the JSON Schema validators for all validated collections.
func AllSchemas() []CollectionSchema {
	return []CollectionSchema{
		usersSchema(),
		tenantsSchema(),
		// Procurement collections
		suppliersSchema(),
		supplierCodesSchema(),
		goodsGroupsSchema(),
		productsSchema(),
		freightCarriersSchema(),
		harboursSchema(),
		containersSchema(),
		countriesSchema(),
		ordersSchema(),
		orderTasksSchema(),
		orderPaymentsSchema(),
		offersSchema(),
		tenantMembershipsSchema(),
		invitationsSchema(),
		plansSchema(),
		creditBundlesSchema(),
		financialTransactionsSchema(),
		webhooksSchema(),
		apiKeysSchema(),
		configVarsSchema(),
		announcementsSchema(),
		customPagesSchema(),
		messagesSchema(),
		usageEventsSchema(),
		ssoConnectionsSchema(),
		eventDefinitionsSchema(),
	}
}

// EnsureSchemaValidation applies JSON Schema validators to all validated
// collections using collMod with moderate validation level.
func (m *MongoDB) EnsureSchemaValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, cs := range AllSchemas() {
		// Ensure the collection exists (ignore "already exists" errors).
		_ = m.Database.CreateCollection(ctx, cs.Collection)

		cmd := bson.D{
			{Key: "collMod", Value: cs.Collection},
			{Key: "validator", Value: cs.Schema},
			{Key: "validationLevel", Value: "moderate"},
			{Key: "validationAction", Value: "error"},
		}

		if err := m.Database.RunCommand(ctx, cmd).Err(); err != nil {
			slog.Warn("failed to apply schema validation", "collection", cs.Collection, "error", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Individual collection schemas
// ---------------------------------------------------------------------------

func usersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "users",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"email", "displayName", "authMethods", "createdAt", "updatedAt"},
				"properties": bson.M{
					"email": bson.M{
						"bsonType": "string",
					},
					"displayName": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"authMethods": bson.M{
						"bsonType": "array",
						"minItems": 1,
						"items": bson.M{
							"bsonType": "string",
							"enum":     bson.A{"password", "google", "github", "microsoft", "magic_link", "passkey"},
						},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
					"emailVerified": bson.M{
						"bsonType": "bool",
					},
					"isActive": bson.M{
						"bsonType": "bool",
					},
					"themePreference": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"light", "dark", "system", ""},
					},
				},
			},
		},
	}
}

func tenantsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "tenants",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "slug", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"slug": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
					"isRoot": bson.M{
						"bsonType": "bool",
					},
					"isActive": bson.M{
						"bsonType": "bool",
					},
					"billingStatus": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"none", "active", "past_due", "canceled", ""},
					},
					"seatQuantity": bson.M{
						"bsonType": "int",
					},
				},
			},
		},
	}
}

func tenantMembershipsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "tenant_memberships",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"userId", "tenantId", "role", "joinedAt", "updatedAt"},
				"properties": bson.M{
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"role": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"owner", "admin", "user"},
					},
					"joinedAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func invitationsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "invitations",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "email", "role", "token", "status", "invitedBy", "expiresAt", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"email": bson.M{
						"bsonType": "string",
					},
					"role": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"owner", "admin", "user"},
					},
					"token": bson.M{
						"bsonType": "string",
					},
					"status": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"pending", "accepted"},
					},
					"invitedBy": bson.M{
						"bsonType": "objectId",
					},
					"expiresAt": bson.M{
						"bsonType": "date",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func plansSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "plans",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "pricingModel", "creditResetPolicy", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"pricingModel": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"flat", "per_seat"},
					},
					"creditResetPolicy": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"reset", "accrue"},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
					"monthlyPriceCents": bson.M{
						"bsonType": "long",
						"minimum":  0,
					},
					"annualDiscountPct": bson.M{
						"bsonType": "int",
						"minimum":  0,
						"maximum":  100,
					},
					"trialDays": bson.M{
						"bsonType": "int",
						"minimum":  0,
					},
				},
			},
		},
	}
}

func creditBundlesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "credit_bundles",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "credits", "priceCents", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"credits": bson.M{
						"bsonType": "long",
						"minimum":  1,
					},
					"priceCents": bson.M{
						"bsonType": "long",
						"minimum":  1,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func financialTransactionsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "financial_transactions",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "type", "currency", "invoiceNumber", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"type": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"subscription", "credit_purchase", "refund"},
					},
					"currency": bson.M{
						"bsonType": "string",
					},
					"invoiceNumber": bson.M{
						"bsonType": "string",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func webhooksSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "webhooks",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "url", "secret", "secretPreview", "events", "createdBy", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"url": bson.M{
						"bsonType": "string",
					},
					"secret": bson.M{
						"bsonType": "string",
					},
					"secretPreview": bson.M{
						"bsonType": "string",
					},
					"events": bson.M{
						"bsonType": "array",
						"minItems": 1,
					},
					"createdBy": bson.M{
						"bsonType": "objectId",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func apiKeysSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "api_keys",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "keyHash", "keyPreview", "authority", "createdBy", "createdAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"keyHash": bson.M{
						"bsonType": "string",
					},
					"keyPreview": bson.M{
						"bsonType": "string",
					},
					"authority": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"admin", "user"},
					},
					"createdBy": bson.M{
						"bsonType": "objectId",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func configVarsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "config_vars",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "type", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"type": bson.M{
						"bsonType": "string",
						"enum":     bson.A{"string", "numeric", "enum", "template"},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func announcementsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "announcements",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"title", "body", "createdAt", "updatedAt"},
				"properties": bson.M{
					"title": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"body": bson.M{
						"bsonType":  "string",
						"minLength": 1,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func customPagesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "custom_pages",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"slug", "title", "createdAt", "updatedAt"},
				"properties": bson.M{
					"slug": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"title": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func messagesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "messages",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"userId", "subject", "body", "createdAt"},
				"properties": bson.M{
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"subject": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 200,
					},
					"body": bson.M{
						"bsonType":  "string",
						"minLength": 1,
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func usageEventsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "usage_events",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "userId", "type", "quantity", "createdAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"userId": bson.M{
						"bsonType": "objectId",
					},
					"type": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 100,
					},
					"quantity": bson.M{
						"bsonType": "int",
						"minimum":  1,
					},
					"metadata": bson.M{
						"bsonType": "object",
						"additionalProperties": bson.M{
							"bsonType": "string",
						},
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func ssoConnectionsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "sso_connections",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "idpEntityId", "idpSsoUrl", "idpCertificate", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId": bson.M{
						"bsonType": "objectId",
					},
					"idpEntityId": bson.M{
						"bsonType": "string",
					},
					"idpSsoUrl": bson.M{
						"bsonType": "string",
					},
					"idpCertificate": bson.M{
						"bsonType": "string",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

func eventDefinitionsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "event_definitions",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"name": bson.M{
						"bsonType":  "string",
						"minLength": 1,
						"maxLength": 128,
					},
					"description": bson.M{
						"bsonType":  "string",
						"maxLength": 256,
					},
					"parentId": bson.M{
						"bsonType": "objectId",
					},
					"createdAt": bson.M{
						"bsonType": "date",
					},
					"updatedAt": bson.M{
						"bsonType": "date",
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Procurement schemas
// ---------------------------------------------------------------------------

func suppliersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "suppliers",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "company", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId": bson.M{"bsonType": "objectId"},
					"company":  bson.M{"bsonType": "string", "minLength": 1, "maxLength": 200},
					"email":    bson.M{"bsonType": "string", "maxLength": 120},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func supplierCodesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "supplier_codes",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "supplierId", "short", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":   bson.M{"bsonType": "objectId"},
					"supplierId": bson.M{"bsonType": "objectId"},
					"short":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 10},
					"createdAt":  bson.M{"bsonType": "date"},
					"updatedAt":  bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func goodsGroupsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "goods_groups",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"name":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 100},
					"short":     bson.M{"bsonType": "string", "maxLength": 100},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func productsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "products",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "nameShort", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"nameShort": bson.M{"bsonType": "string", "minLength": 1, "maxLength": 45},
					"ean":       bson.M{"bsonType": "string", "maxLength": 32},
					"wtn":       bson.M{"bsonType": "string", "maxLength": 30},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func freightCarriersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "freight_carriers",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"name":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 50},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func harboursSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "harbours",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"name":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 50},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func containersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "containers",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"name":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 50},
					"volumeM3":  bson.M{"bsonType": "double", "minimum": 0},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func countriesSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "countries",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "name", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"name":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 50},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func ordersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "orders",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "orderNumber", "orderDate", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":     bson.M{"bsonType": "objectId"},
					"orderNumber":  bson.M{"bsonType": "string", "minLength": 1, "maxLength": 64},
					"orderDate":    bson.M{"bsonType": "date"},
					"orderSumUsd":  bson.M{"bsonType": "double", "minimum": 0},
					"createdAt":    bson.M{"bsonType": "date"},
					"updatedAt":    bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func orderTasksSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "order_tasks",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "orderId", "text", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"orderId":   bson.M{"bsonType": "objectId"},
					"text":      bson.M{"bsonType": "string", "minLength": 1, "maxLength": 500},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func orderPaymentsSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "order_payments",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "orderId", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":          bson.M{"bsonType": "objectId"},
					"orderId":           bson.M{"bsonType": "objectId"},
					"paymentAmountEur":  bson.M{"bsonType": "double", "minimum": 0},
					"paymentDollarRate": bson.M{"bsonType": "double", "minimum": 0},
					"paymentFees":       bson.M{"bsonType": "double", "minimum": 0},
					"createdAt":         bson.M{"bsonType": "date"},
					"updatedAt":         bson.M{"bsonType": "date"},
				},
			},
		},
	}
}

func offersSchema() CollectionSchema {
	return CollectionSchema{
		Collection: "offers",
		Schema: bson.M{
			"$jsonSchema": bson.M{
				"bsonType": "object",
				"required": bson.A{"tenantId", "createdAt", "updatedAt"},
				"properties": bson.M{
					"tenantId":  bson.M{"bsonType": "objectId"},
					"priceUsd":  bson.M{"bsonType": "double", "minimum": 0},
					"createdAt": bson.M{"bsonType": "date"},
					"updatedAt": bson.M{"bsonType": "date"},
				},
			},
		},
	}
}
