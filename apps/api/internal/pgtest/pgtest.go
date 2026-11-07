// Package pgtest gives a test its own Postgres schema, migrated with the
// embedded migrations. It exists so a test can prove a query against the real
// DDL; with TEST_DATABASE_URL unset the test is skipped.
package pgtest

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New returns a handle on a fresh, migrated schema that is dropped when the
// test ends, or skips the test when TEST_DATABASE_URL is unset.
func New(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	schema := fmt.Sprintf("t_%d", time.Now().UnixNano())
	admin := open(t, dsn)
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	scoped := open(t, withSearchPath(dsn, schema))
	sqlDB, err := scoped.DB()
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
	})
	err = migrate.Run(context.Background(), migrate.Config{
		DB: sqlDB, FS: os.DirFS(migrationsDir()), Logger: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatalf("migrations: %v", err)
	}
	return scoped
}

func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

func open(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db
}

func withSearchPath(dsn, schema string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
