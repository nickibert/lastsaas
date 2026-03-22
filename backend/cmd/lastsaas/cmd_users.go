package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"lastsaas/internal/auth"
	"lastsaas/internal/db"
	"lastsaas/internal/models"
	"lastsaas/internal/validation"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func cmdUsers() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, `Usage: lastsaas users <subcommand>

Subcommands:
  list                          List all users
  get --email <email>           Show user details
  create --email <email> --name <name> [--role owner|admin|user] [--tenant <id>]
                                Create a new user with password
  suspend --email <email>       Suspend a user account
  activate --email <email>      Reactivate a suspended account
  revoke-sessions --email <email>  Revoke all sessions for a user`)
		os.Exit(1)
	}

	switch os.Args[2] {
	case "list":
		cmdUsersList()
	case "get":
		cmdUsersGet()
	case "create":
		cmdUsersCreate()
	case "suspend":
		cmdUsersSetActive(false)
	case "activate":
		cmdUsersSetActive(true)
	case "revoke-sessions":
		cmdUsersRevokeSessions()
	default:
		fmt.Fprintf(os.Stderr, "Unknown users subcommand: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func cmdUsersList() {
	fs := flag.NewFlagSet("users list", flag.ExitOnError)
	limit := fs.Int("limit", 50, "Max users to show")
	inactive := fs.Bool("inactive", false, "Show only inactive/suspended users")
	fs.Parse(os.Args[3:])

	database, _, cleanup := connectDB()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	filter := bson.M{}
	if *inactive {
		filter["isActive"] = false
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(int64(*limit))

	cursor, err := database.Users().Find(ctx, filter, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to query users: %v\n", err)
		os.Exit(1)
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read users: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		type userRow struct {
			ID          string   `json:"id"`
			Email       string   `json:"email"`
			DisplayName string   `json:"displayName"`
			IsActive    bool     `json:"isActive"`
			MFA         bool     `json:"mfa"`
			AuthMethods []string `json:"authMethods"`
			CreatedAt   string   `json:"createdAt"`
			LastLogin   string   `json:"lastLogin,omitempty"`
		}
		rows := make([]userRow, 0, len(users))
		for _, u := range users {
			methods := make([]string, len(u.AuthMethods))
			for i, m := range u.AuthMethods {
				methods[i] = string(m)
			}
			r := userRow{
				ID:          u.ID.Hex(),
				Email:       u.Email,
				DisplayName: u.DisplayName,
				IsActive:    u.IsActive,
				MFA:         u.TOTPEnabled,
				AuthMethods: methods,
				CreatedAt:   u.CreatedAt.Format(time.RFC3339),
			}
			if u.LastLoginAt != nil {
				r.LastLogin = u.LastLoginAt.Format(time.RFC3339)
			}
			rows = append(rows, r)
		}
		printJSON(rows)
		return
	}

	if len(users) == 0 {
		fmt.Println("No users found.")
		return
	}

	fmt.Printf("%-36s %-30s %-20s %-8s %-5s %s\n",
		bold("ID"), bold("EMAIL"), bold("NAME"), bold("STATUS"), bold("MFA"), bold("CREATED"))
	fmt.Printf("%-36s %-30s %-20s %-8s %-5s %s\n",
		"----", "-----", "----", "------", "---", "-------")

	for _, u := range users {
		status := clr(cGreen, "active")
		if !u.IsActive {
			status = clr(cRed, "suspended")
		}
		mfa := ""
		if u.TOTPEnabled {
			mfa = clr(cGreen, "yes")
		}
		fmt.Printf("%-36s %-30s %-20s %-8s %-5s %s\n",
			u.ID.Hex(),
			truncate(u.Email, 30),
			truncate(u.DisplayName, 20),
			status,
			mfa,
			timeAgo(u.CreatedAt),
		)
	}
	fmt.Printf("\n%d users shown\n", len(users))
}

func cmdUsersCreate() {
	fs := flag.NewFlagSet("users create", flag.ExitOnError)
	email := fs.String("email", "", "Email address (required)")
	name := fs.String("name", "", "Display name (required)")
	role := fs.String("role", "admin", "Role: owner, admin, or user (default: admin)")
	tenantID := fs.String("tenant", "", "Tenant ID to add membership (default: root tenant)")
	fs.Parse(os.Args[3:])

	if *email == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "Usage: lastsaas users create --email <email> --name <name> [--role owner|admin|user] [--tenant <id>]")
		os.Exit(1)
	}

	memberRole := models.MemberRole(*role)
	if !models.ValidRole(memberRole) {
		fmt.Fprintf(os.Stderr, "Invalid role %q — must be owner, admin, or user\n", *role)
		os.Exit(1)
	}

	database, _, cleanup := connectDB()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	emailNorm := strings.TrimSpace(strings.ToLower(*email))

	// Check for duplicate
	var existing models.User
	if err := database.Users().FindOne(ctx, bson.M{"email": emailNorm}).Decode(&existing); err == nil {
		fmt.Fprintf(os.Stderr, "User already exists: %s\n", emailNorm)
		os.Exit(1)
	}

	// Resolve tenant
	var tenant models.Tenant
	if *tenantID != "" {
		tid, err := primitive.ObjectIDFromHex(*tenantID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid tenant ID: %s\n", *tenantID)
			os.Exit(1)
		}
		if err := database.Tenants().FindOne(ctx, bson.M{"_id": tid}).Decode(&tenant); err != nil {
			fmt.Fprintf(os.Stderr, "Tenant not found: %s\n", *tenantID)
			os.Exit(1)
		}
	} else {
		if err := database.Tenants().FindOne(ctx, bson.M{"isRoot": true}).Decode(&tenant); err != nil {
			fmt.Fprintln(os.Stderr, "Root tenant not found — run 'lastsaas setup' first")
			os.Exit(1)
		}
	}

	passwordService := auth.NewPasswordService()
	password := promptPassword("Password")
	confirm := promptPassword("Confirm password")

	if password != confirm {
		fmt.Fprintln(os.Stderr, "Passwords do not match.")
		os.Exit(1)
	}

	if err := passwordService.ValidatePasswordStrength(password); err != nil {
		fmt.Fprintf(os.Stderr, "Password too weak: %v\n", err)
		fmt.Fprintln(os.Stderr, "Requirements: 10+ characters, uppercase, lowercase, number, special character")
		os.Exit(1)
	}

	passwordHash, err := passwordService.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to hash password: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()

	user := models.User{
		ID:            primitive.NewObjectID(),
		Email:         emailNorm,
		DisplayName:   strings.TrimSpace(*name),
		PasswordHash:  passwordHash,
		AuthMethods:   []models.AuthMethod{models.AuthMethodPassword},
		EmailVerified: true,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := validation.Validate(&user); err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}
	if _, err := database.Users().InsertOne(ctx, user); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create user: %v\n", err)
		os.Exit(1)
	}

	membership := models.TenantMembership{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		TenantID:  tenant.ID,
		Role:      memberRole,
		JoinedAt:  now,
		UpdatedAt: now,
	}
	if err := validation.Validate(&membership); err != nil {
		database.Users().DeleteOne(ctx, bson.M{"_id": user.ID})
		fmt.Fprintf(os.Stderr, "Membership validation failed: %v\n", err)
		os.Exit(1)
	}
	if _, err := database.TenantMemberships().InsertOne(ctx, membership); err != nil {
		database.Users().DeleteOne(ctx, bson.M{"_id": user.ID})
		fmt.Fprintf(os.Stderr, "Failed to create membership: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User created successfully.\n")
	fmt.Printf("  ID:     %s\n", user.ID.Hex())
	fmt.Printf("  Email:  %s\n", user.Email)
	fmt.Printf("  Name:   %s\n", user.DisplayName)
	fmt.Printf("  Role:   %s\n", memberRole)
	fmt.Printf("  Tenant: %s (%s)\n", tenant.Name, tenant.ID.Hex())
}

func cmdUsersGet() {
	fs := flag.NewFlagSet("users get", flag.ExitOnError)
	email := fs.String("email", "", "User email address (required)")
	fs.Parse(os.Args[3:])

	if *email == "" {
		fmt.Fprintln(os.Stderr, "Usage: lastsaas users get --email <email>")
		os.Exit(1)
	}

	database, _, cleanup := connectDB()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	user, memberships := lookupUserWithMemberships(ctx, database, *email)

	// Resolve tenant names
	tenantIDs := make([]primitive.ObjectID, 0, len(memberships))
	for _, m := range memberships {
		tenantIDs = append(tenantIDs, m.TenantID)
	}
	tenantNames := resolveTenantNames(ctx, database, tenantIDs)

	if jsonOutput {
		type membershipInfo struct {
			TenantID   string `json:"tenantId"`
			TenantName string `json:"tenantName"`
			Role       string `json:"role"`
		}
		type detail struct {
			ID          string           `json:"id"`
			Email       string           `json:"email"`
			DisplayName string           `json:"displayName"`
			IsActive    bool             `json:"isActive"`
			MFA         bool             `json:"mfa"`
			AuthMethods []string         `json:"authMethods"`
			Verified    bool             `json:"emailVerified"`
			Memberships []membershipInfo `json:"memberships"`
			CreatedAt   string           `json:"createdAt"`
			LastLogin   string           `json:"lastLogin,omitempty"`
		}
		methods := make([]string, len(user.AuthMethods))
		for i, m := range user.AuthMethods {
			methods[i] = string(m)
		}
		d := detail{
			ID:          user.ID.Hex(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
			IsActive:    user.IsActive,
			MFA:         user.TOTPEnabled,
			AuthMethods: methods,
			Verified:    user.EmailVerified,
			CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		}
		if user.LastLoginAt != nil {
			d.LastLogin = user.LastLoginAt.Format(time.RFC3339)
		}
		for _, m := range memberships {
			d.Memberships = append(d.Memberships, membershipInfo{
				TenantID:   m.TenantID.Hex(),
				TenantName: tenantNames[m.TenantID],
				Role:       string(m.Role),
			})
		}
		printJSON(d)
		return
	}

	fmt.Printf("%s %s\n", bold("User:"), user.DisplayName)
	fmt.Printf("  ID:         %s\n", user.ID.Hex())
	fmt.Printf("  Email:      %s\n", user.Email)
	fmt.Printf("  Verified:   %v\n", user.EmailVerified)
	status := clr(cGreen, "active")
	if !user.IsActive {
		status = clr(cRed, "suspended")
	}
	fmt.Printf("  Status:     %s\n", status)

	methods := make([]string, len(user.AuthMethods))
	for i, m := range user.AuthMethods {
		methods[i] = string(m)
	}
	fmt.Printf("  Auth:       %s\n", strings.Join(methods, ", "))
	if user.TOTPEnabled {
		fmt.Printf("  MFA:        %s\n", clr(cGreen, "enabled"))
	}
	fmt.Printf("  Created:    %s (%s)\n", user.CreatedAt.Format(time.RFC3339), timeAgo(user.CreatedAt))
	if user.LastLoginAt != nil {
		fmt.Printf("  Last login: %s (%s)\n", user.LastLoginAt.Format(time.RFC3339), timeAgo(*user.LastLoginAt))
	}

	if len(memberships) > 0 {
		fmt.Printf("\n  %s\n", bold("Memberships:"))
		for _, m := range memberships {
			name := tenantNames[m.TenantID]
			if name == "" {
				name = m.TenantID.Hex()
			}
			fmt.Printf("    - %s (%s)\n", name, m.Role)
		}
	}
}

func cmdUsersSetActive(active bool) {
	verb := "suspend"
	if active {
		verb = "activate"
	}

	fs := flag.NewFlagSet("users "+verb, flag.ExitOnError)
	email := fs.String("email", "", "User email address (required)")
	fs.Parse(os.Args[3:])

	if *email == "" {
		fmt.Fprintf(os.Stderr, "Usage: lastsaas users %s --email <email>\n", verb)
		os.Exit(1)
	}

	database, _, cleanup := connectDB()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	emailNorm := strings.TrimSpace(strings.ToLower(*email))
	var user models.User
	if err := database.Users().FindOne(ctx, bson.M{"email": emailNorm}).Decode(&user); err != nil {
		fmt.Fprintf(os.Stderr, "User not found: %s\n", emailNorm)
		os.Exit(1)
	}

	if user.IsActive == active {
		if active {
			fmt.Printf("User %s is already active.\n", emailNorm)
		} else {
			fmt.Printf("User %s is already suspended.\n", emailNorm)
		}
		return
	}

	_, err := database.Users().UpdateOne(ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"isActive": active, "updatedAt": time.Now()}},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update user: %v\n", err)
		os.Exit(1)
	}

	if active {
		fmt.Printf("User %s (%s) has been reactivated.\n", user.DisplayName, user.Email)
	} else {
		// Also revoke sessions when suspending
		database.RefreshTokens().DeleteMany(ctx, bson.M{"userId": user.ID})
		fmt.Printf("User %s (%s) has been suspended and all sessions revoked.\n", user.DisplayName, user.Email)
	}
}

func cmdUsersRevokeSessions() {
	fs := flag.NewFlagSet("users revoke-sessions", flag.ExitOnError)
	email := fs.String("email", "", "User email address (required)")
	fs.Parse(os.Args[3:])

	if *email == "" {
		fmt.Fprintln(os.Stderr, "Usage: lastsaas users revoke-sessions --email <email>")
		os.Exit(1)
	}

	database, _, cleanup := connectDB()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	emailNorm := strings.TrimSpace(strings.ToLower(*email))
	var user models.User
	if err := database.Users().FindOne(ctx, bson.M{"email": emailNorm}).Decode(&user); err != nil {
		fmt.Fprintf(os.Stderr, "User not found: %s\n", emailNorm)
		os.Exit(1)
	}

	result, err := database.RefreshTokens().DeleteMany(ctx, bson.M{"userId": user.ID})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to revoke sessions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Revoked %d session(s) for %s (%s).\n", result.DeletedCount, user.DisplayName, user.Email)
}

// lookupUserWithMemberships finds a user by email and their memberships.
func lookupUserWithMemberships(ctx context.Context, database *db.MongoDB, email string) (models.User, []models.TenantMembership) {
	emailNorm := strings.TrimSpace(strings.ToLower(email))
	var user models.User
	if err := database.Users().FindOne(ctx, bson.M{"email": emailNorm}).Decode(&user); err != nil {
		fmt.Fprintf(os.Stderr, "User not found: %s\n", emailNorm)
		os.Exit(1)
	}

	cursor, err := database.TenantMemberships().Find(ctx, bson.M{"userId": user.ID})
	if err != nil {
		return user, nil
	}
	defer cursor.Close(ctx)

	var memberships []models.TenantMembership
	cursor.All(ctx, &memberships)
	return user, memberships
}

// resolveTenantNames batch-resolves tenant IDs to names.
func resolveTenantNames(ctx context.Context, database *db.MongoDB, ids []primitive.ObjectID) map[primitive.ObjectID]string {
	names := make(map[primitive.ObjectID]string)
	if len(ids) == 0 {
		return names
	}
	cursor, err := database.Tenants().Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return names
	}
	defer cursor.Close(ctx)

	var tenants []models.Tenant
	cursor.All(ctx, &tenants)
	for _, t := range tenants {
		names[t.ID] = t.Name
	}
	return names
}
