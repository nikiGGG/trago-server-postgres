package database_test

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jinzhu/gorm/dialects/sqlite" // register the "sqlite3" driver for conflict fixtures
	_ "github.com/lib/pq"                      // register the "postgres" driver for maintenance connections
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/database"
	"github.com/traggo/server/model"
	"github.com/traggo/server/test"
)

// ---- SQLite migration-failure fixture (T011) ----

// TestMigrationFailure_sqlite proves a real, post-connect schema conflict on an
// isolated SQLite database makes public database.New fail: a view named users
// blocks CREATE TABLE users during AutoMigrate. The failure must occur during
// migration, not during connection opening, and New must not return a handle.
func TestMigrationFailure_sqlite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conflict.db")

	// Connect successfully, then introduce a deterministic schema conflict: a
	// view named "users". GORM's SQLite HasTable only matches sqlite_master
	// rows with type='table', so AutoMigrate tries CREATE TABLE users and the
	// pre-existing view makes that statement fail.
	sqlDB, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`CREATE VIEW users AS SELECT 1 AS id`)
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	db, err := database.New("sqlite3", path)
	require.Nil(t, db, "failed migration must not return a usable handle")
	require.Error(t, err, "migration conflict must be propagated from public New")
	assert.Contains(t, err.Error(), "migrate database schema")
	// The connection opened fine; the failure must be the schema conflict.
	assert.Contains(t, err.Error(), "already exists", "failure must be the CREATE TABLE conflict")
}

// ---- PostgreSQL migration-failure fixture (T011) ----
//
// These live in the external database_test package on purpose. Internal
// database tests importing the test harness would form an import cycle
// (test -> database), so the public-New PostgreSQL regressions live here and
// reuse the harness's disposable-server configuration.

// skipOrFailPostgres returns the disposable PostgreSQL config, skipping when no
// DSN is set and failing (never skipping) when the DSN is required.
func skipOrFailPostgres(t *testing.T) test.PostgresConfig {
	t.Helper()
	cfg := test.PostgresConfigFromEnv()
	if !cfg.Enabled() {
		if cfg.Required {
			t.Fatalf("%s not set but %s=1: PostgreSQL tests are required and cannot run",
				test.PostgresDSNEnv, test.PostgresRequiredEnv)
		}
		t.Skipf("%s not set; skipping PostgreSQL test", test.PostgresDSNEnv)
	}
	return cfg
}

// createEmptyDisposablePostgres creates a brand-new isolated database and
// returns its name and a DSN that targets it. The database is dropped at test
// cleanup. No schema is created, so the caller controls the conflict fixture.
func createEmptyDisposablePostgres(t *testing.T, cfg test.PostgresConfig) (dbName, targetDSN string) {
	t.Helper()
	dbName = fmt.Sprintf("traggo_migfail_%d_%d", os.Getpid(), time.Now().UnixNano())

	admin, err := sql.Open("postgres", cfg.BaseDSN)
	require.NoError(t, err, "open maintenance connection")
	require.NoError(t, admin.Ping(), "ping disposable PostgreSQL server")
	_, err = admin.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName))
	require.NoError(t, admin.Close())
	require.NoError(t, err, "create disposable database")

	targetDSN, err = rewritePostgresDSN(cfg.BaseDSN, dbName)
	require.NoError(t, err)

	t.Cleanup(func() { dropDisposablePostgres(cfg.BaseDSN, dbName) })
	return dbName, targetDSN
}

// rewritePostgresDSN rewrites a lib/pq connection string so it targets the
// given database. Both URL (postgres://) and keyword/value forms are supported.
func rewritePostgresDSN(dsn, dbName string) (string, error) {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(trimmed, "postgres://") || strings.HasPrefix(trimmed, "postgresql://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("parse postgres URL DSN: %w", err)
		}
		u.Path = "/" + dbName
		return u.String(), nil
	}
	fields := strings.Fields(trimmed)
	replaced := false
	for i, f := range fields {
		if strings.HasPrefix(f, "dbname=") {
			fields[i] = "dbname=" + dbName
			replaced = true
		}
	}
	if !replaced {
		fields = append(fields, "dbname="+dbName)
	}
	return strings.Join(fields, " "), nil
}

func dropDisposablePostgres(baseDSN, dbName string) {
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return
	}
	defer admin.Close()
	_, _ = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %q WITH (FORCE)", dbName))
}

// TestMigrationFailure_postgres is the regression WP04 registers in its expected
// live-suite manifest. It connects to the configured disposable server and
// executes a real AutoMigrate failure (a view named users blocks CREATE TABLE
// users), not merely a malformed/unusable DSN.
func TestMigrationFailure_postgres(t *testing.T) {
	cfg := skipOrFailPostgres(t)
	_, targetDSN := createEmptyDisposablePostgres(t, cfg)

	// Successfully connect, then introduce a deterministic schema conflict: a
	// view named "users". GORM's PostgreSQL HasTable only matches rows with
	// table_type='BASE TABLE', so AutoMigrate tries CREATE TABLE users and the
	// pre-existing view makes that statement fail.
	sqlDB, err := sql.Open("postgres", targetDSN)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`CREATE VIEW users AS SELECT 1 AS id`)
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	db, err := database.New("postgres", targetDSN)
	require.Nil(t, db, "failed migration must not return a usable handle")
	require.Error(t, err, "migration conflict must be propagated from public New")
	assert.Contains(t, err.Error(), "migrate database schema")
	// The connection opened fine; the failure must be the schema conflict.
	assert.Contains(t, err.Error(), "already exists", "failure must be the CREATE TABLE conflict")
}

// ---- Healthy fresh/reopen on PostgreSQL (T015) ----

// TestInitializationReopen_postgres proves production New initializes a fresh
// PostgreSQL schema and cleanly reopens an already-migrated one, reading and
// writing data in both cases. WP04 requires this live-server execution
// separately from the migration-error case.
func TestInitializationReopen_postgres(t *testing.T) {
	cfg := skipOrFailPostgres(t)

	p := test.NewPostgresDB(t, cfg)

	// Fresh schema created by production New (see test.NewPostgresDB): write
	// and read a user to prove the schema is usable.
	u := model.User{Name: "reopen", Pass: []byte("secret")}
	require.NoError(t, p.Create(&u).Error, "write after fresh initialization")
	var fresh model.User
	require.NoError(t, p.First(&fresh, "name = ?", "reopen").Error)
	require.Equal(t, []byte("secret"), fresh.Pass, "read back after fresh initialization")

	dbName := p.DatabaseName
	// Close the connection but keep the database alive so we can reopen it.
	require.NoError(t, p.DB.Close())
	t.Cleanup(func() { dropDisposablePostgres(cfg.BaseDSN, dbName) })

	// Reopen the already-migrated database through production New: a healthy
	// reopen must not fail and the existing data must remain readable.
	targetDSN, err := rewritePostgresDSN(cfg.BaseDSN, dbName)
	require.NoError(t, err)
	reopened, err := database.New("postgres", targetDSN)
	require.NoError(t, err, "reopening an already-migrated schema must succeed")
	defer reopened.Close()

	var after model.User
	require.NoError(t, reopened.First(&after, "name = ?", "reopen").Error)
	require.Equal(t, "reopen", after.Name)
	require.Equal(t, []byte("secret"), after.Pass, "read back after reopen")

	// The reopened schema is still writable.
	u2 := model.User{Name: "reopen-2", Pass: []byte("more")}
	require.NoError(t, reopened.Create(&u2).Error, "write after reopen")
	var count int
	require.NoError(t, reopened.Model(&model.User{}).Count(&count).Error)
	require.Equal(t, 2, count)
}
