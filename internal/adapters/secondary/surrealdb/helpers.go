package surrealdb

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
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
	switch v := id.ID.(type) {
	case models.UUID:
		return uuid.UUID(v.UUID)
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil
		}
		return parsed
	default:
		parsed, err := uuid.Parse(fmt.Sprintf("%v", v))
		if err != nil {
			return uuid.Nil
		}
		return parsed
	}
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
