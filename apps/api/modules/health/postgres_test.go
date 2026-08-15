package health

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/migrate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// testDB is the scoped handle every test in this package shares, and skipReason
// is why there is none. Nothing here runs against SQLite: the schema under test
// is the Postgres DDL in apps/api/migrations, jsonb and the cascade included,
// and a different engine would prove nothing about it.
var (
	testDB     *gorm.DB
	skipReason string
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		// Loud on stderr as well as skipped: `go test` without -v shows a skip
		// nowhere, and a database test that quietly does not run is worse than
		// one that fails.
		skipReason = "TEST_DATABASE_URL is not set, so the health store was NOT exercised against Postgres"
		fmt.Fprintf(os.Stderr, "health: %s\n", skipReason)
		os.Exit(m.Run())
	}

	schema := fmt.Sprintf("health_test_%d", time.Now().UnixNano())
	admin, err := open(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health: TEST_DATABASE_URL is set but unusable: %v\n", err)
		os.Exit(1)
	}
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		fmt.Fprintf(os.Stderr, "health: could not create the scratch schema: %v\n", err)
		os.Exit(1)
	}

	scoped, err := open(withSearchPath(dsn, schema))
	if err != nil {
		fmt.Fprintf(os.Stderr, "health: could not open the scratch schema: %v\n", err)
		os.Exit(1)
	}
	sqlDB, err := scoped.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "health: no database handle: %v\n", err)
		os.Exit(1)
	}
	if err := migrate.Run(context.Background(), migrate.Config{
		DB:     sqlDB,
		FS:     os.DirFS("../../migrations"),
		Logger: slog.New(slog.DiscardHandler),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "health: migrations failed: %v\n", err)
		os.Exit(1)
	}

	testDB = scoped
	code := m.Run()

	if err := sqlDB.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "health: closing the test connection failed: %v\n", err)
	}
	if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
		fmt.Fprintf(os.Stderr, "health: dropping schema %s failed, it is now orphaned: %v\n", schema, err)
	}
	os.Exit(code)
}

func open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:  gormlogger.Discard,
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
}

func withSearchPath(dsn, schema string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return dsn + " search_path=" + schema
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func store(t *testing.T) (*Store, context.Context) {
	t.Helper()
	if testDB == nil {
		t.Skip(skipReason)
	}
	return NewStore(testDB), context.Background()
}

// target creates the delivery target a health row hangs off, because the foreign
// key means health for a target that does not exist cannot be written at all.
func target(t *testing.T, id string) string {
	t.Helper()
	err := testDB.Exec(
		"INSERT INTO delivery_targets (id, type, name) VALUES (?, 'webhook', ?)", id, id).Error
	if err != nil {
		t.Fatalf("could not create the delivery target: %v", err)
	}
	t.Cleanup(func() { testDB.Exec("DELETE FROM delivery_targets WHERE id = ?", id) })
	return id
}

func moment(offset time.Duration) *time.Time {
	// Postgres keeps microseconds, so a nanosecond here would round-trip unequal
	// for reasons that have nothing to do with this store.
	value := time.Now().UTC().Add(offset).Truncate(time.Microsecond)
	return &value
}
