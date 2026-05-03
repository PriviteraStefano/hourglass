package surrealdb

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
	"github.com/fxamacker/cbor/v2"
)

func uuidToRecordID(table string, id uuid.UUID) models.RecordID {
	sdkUUID := models.UUID{UUID: gofrsUUID(id)}
	return models.NewRecordID(table, sdkUUID)
}

func gofrsUUID(u uuid.UUID) [16]byte {
	var arr [16]byte
	copy(arr[:], u[:])
	return arr
}

func recordIDToUUID(id models.RecordID) uuid.UUID {
	if id.ID == nil {
		return uuid.Nil
	}

	// Most common case: CBOR-encoded UUID
	if tag, ok := id.ID.(cbor.Tag); ok && len(tag.Content.([]byte)) == 16 {
		uuidBA := tag.Content.([]byte)
		return uuid.UUID(uuidBA)
	}

	// Fallback for string representation
	if str, ok := id.ID.(string); ok {
		u, err := uuid.Parse(str)
		if err != nil {
			return uuid.Nil
		}
		return u
	}

	return uuid.Nil
}


func recordIDToUserID(id *models.RecordID) string {
	if id == nil {
		return ""
	}
	switch v := id.ID.(type) {
	case models.UUID:
		return uuid.UUID(v.UUID).String()
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func recordIDToUUIDPtr(id models.RecordID) *uuid.UUID {
	if id.ID == nil {
		return nil
	}
	result := recordIDToUUID(id)
	if result == uuid.Nil {
		return nil
	}
	return &result
}

func uuidToRecordIDPtr(table string, id *uuid.UUID) models.RecordID {
	if id == nil {
		return models.RecordID{}
	}
	return uuidToRecordID(table, *id)
}

func wrapErr(err error, op string) error {
	if err == nil {
		return nil
	}
	if isNotFound(err) {
		return ports.ErrUserNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "not found") ||
		contains(errStr, "record not found") ||
		contains(errStr, "No record found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

var dbInstance *sdb.DB

func InitDB() error {
	if dbInstance != nil {
		return nil
	}
	ctx := context.Background()
	url := getEnvOrDefault("SURREALDB_URL", "ws://localhost:8000/rpc")
	user := getEnvOrDefault("SURREALDB_USER", "root")
	pass := getEnvOrDefault("SURREALDB_PASS", "root")
	ns := getEnvOrDefault("SURREALDB_NS", "hourglass")
	dbName := getEnvOrDefault("SURREALDB_DB", "main")

	var err error
	dbInstance, err = sdb.FromEndpointURLString(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to connect to SurrealDB: %w", err)
	}

	_, err = dbInstance.SignIn(ctx, &sdb.Auth{
		Username: user,
		Password: pass,
	})
	if err != nil {
		return fmt.Errorf("failed to sign in to SurrealDB: %w", err)
	}

	if err := dbInstance.Use(ctx, ns, dbName); err != nil {
		return fmt.Errorf("failed to use namespace/database: %w", err)
	}

	return nil
}

func GetDB() *sdb.DB {
	if dbInstance == nil {
		if err := InitDB(); err != nil {
			panic(err)
		}
	}
	return dbInstance
}

func CloseDB() error {
	if dbInstance != nil {
		return dbInstance.Close(context.Background())
	}
	return nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func GetTestDBWithNamespace(ns, dbName string) (*sdb.DB, error) {
	ctx := context.Background()

	user := getEnvOrDefault("SURREALDB_USER", "root")
	pass := getEnvOrDefault("SURREALDB_PASS", "root")
	url := os.Getenv("SURREALDB_URL")
	if url == "" {
		url = "ws://localhost:8000/rpc"
	}

	db, err := sdb.FromEndpointURLString(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to create test DB: %w", err)
	}

	if _, err := db.SignIn(ctx, &sdb.Auth{
		Username: user,
		Password: pass,
	}); err != nil {
		db.Close(ctx)
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}

	if err := db.Use(ctx, ns, dbName); err != nil {
		db.Close(ctx)
		return nil, fmt.Errorf("failed to use namespace: %w", err)
	}

	if err := applyTestSchema(ctx, db); err != nil {
		db.Close(ctx)
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	return db, nil
}

func applyTestSchema(ctx context.Context, db *sdb.DB) error {
	queries := []string{
		"DEFINE TABLE users SCHEMAFULL",
		"DEFINE FIELD email ON users TYPE string",
		"DEFINE FIELD username ON users TYPE option<string>",
		"DEFINE FIELD firstname ON users TYPE option<string>",
		"DEFINE FIELD lastname ON users TYPE option<string>",
		"DEFINE FIELD name ON users TYPE string",
		"DEFINE FIELD password_hash ON users TYPE option<string>",
		"DEFINE FIELD is_active ON users TYPE bool DEFAULT true",
		"DEFINE FIELD created_at ON users TYPE datetime",
		"DEFINE FIELD updated_at ON users TYPE datetime",
		"DEFINE INDEX user_email ON users FIELDS email UNIQUE",
		"DEFINE INDEX user_username ON users FIELDS username UNIQUE",
		"DEFINE TABLE organizations SCHEMAFULL",
		"DEFINE FIELD name ON organizations TYPE string",
		"DEFINE FIELD slug ON organizations TYPE string",
		"DEFINE FIELD description ON organizations TYPE option<string>",
		"DEFINE FIELD financial_cutoff_days ON organizations TYPE option<number>",
		"DEFINE FIELD financial_cutoff_config ON organizations TYPE option<object>",
		"DEFINE FIELD financial_cutoff_config.cutoff_day_of_month ON organizations TYPE int",
		"DEFINE FIELD financial_cutoff_config.grace_days ON organizations TYPE int",
		"DEFINE FIELD created_at ON organizations TYPE datetime",
		"DEFINE FIELD updated_at ON organizations TYPE datetime",
		"DEFINE INDEX org_slug ON organizations FIELDS slug UNIQUE",
		// Use string instead of record for user_id
		"DEFINE TABLE refresh_tokens SCHEMAFULL",
		"DEFINE FIELD id ON refresh_tokens TYPE string",
		"DEFINE FIELD user_id ON refresh_tokens TYPE string",
		"DEFINE FIELD token_hash ON refresh_tokens TYPE string",
		"DEFINE FIELD expires_at ON refresh_tokens TYPE datetime",
		"DEFINE FIELD revoked_at ON refresh_tokens TYPE option<datetime>",
		"DEFINE FIELD created_at ON refresh_tokens TYPE datetime",
		"DEFINE INDEX rt_token_hash ON refresh_tokens FIELDS token_hash UNIQUE",
		"DEFINE INDEX rt_user ON refresh_tokens FIELDS user_id",
		"DEFINE TABLE invitations SCHEMAFULL",
		"DEFINE FIELD id ON invitations TYPE string",
		"DEFINE FIELD organization_id ON invitations TYPE string",
		"DEFINE FIELD code ON invitations TYPE string",
		"DEFINE FIELD invite_token ON invitations TYPE string",
		"DEFINE FIELD email ON invitations TYPE option<string>",
		"DEFINE FIELD status ON invitations TYPE string DEFAULT 'pending'",
		"DEFINE FIELD expires_at ON invitations TYPE datetime",
		"DEFINE FIELD created_by ON invitations TYPE string",
		"DEFINE FIELD created_at ON invitations TYPE datetime",
		"DEFINE INDEX invite_code ON invitations FIELDS code UNIQUE",
		"DEFINE INDEX invite_token ON invitations FIELDS invite_token UNIQUE",
	}

	for _, query := range queries {
		_, err := sdb.Query[any](ctx, db, query, nil)
		if err != nil {
			return fmt.Errorf("failed to execute schema query %q: %w", query, err)
		}
	}
	return nil
}
