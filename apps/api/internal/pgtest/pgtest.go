// Package pgtest gives a test its own Postgres schema, migrated with the
// embedded migrations, through tronc/testdb. With TEST_DATABASE_URL unset the
// test is skipped.
package pgtest

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/testdb"
	"gorm.io/gorm"
)

// New returns a handle on a fresh, migrated schema that is dropped when the
// test ends, or skips the test when TEST_DATABASE_URL is unset.
func New(t *testing.T) *gorm.DB {
	t.Helper()
	url, ok := testdb.URL()
	if !ok {
		t.Skip(testdb.SkipReason("a Postgres 16 database"))
	}
	return testdb.OpenIsolated(t, url, testdb.Config{
		Prefix: "antenne_test",
		Migrate: func(db *gorm.DB) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			lock, err := testdb.LockID(db)
			if err != nil {
				return err
			}
			return migrate.Run(context.Background(), migrate.Config{
				DB: sqlDB, FS: os.DirFS(migrationsDir()), Logger: slog.New(slog.DiscardHandler), LockID: lock,
			})
		},
	})
}

func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}
