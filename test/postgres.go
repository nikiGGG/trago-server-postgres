package test

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq" // register the "postgres" driver for the maintenance connection
	"github.com/traggo/server/database"
)

// PostgresDB is a disposable PostgreSQL database used by exactly one test.
//
// Isolation: each PostgresDB is a brand-new database with a random name created
// on the server referenced by BaseDSN. Because a fresh database starts with
// empty tables and fresh sequences, fixed fixture IDs (e.g. user 1) never
// collide with generated sequences and never bleed across tests. The database
// is dropped on Close(). The database named inside BaseDSN is only used as a
// maintenance target for CREATE DATABASE / DROP DATABASE - it never receives
// fixture data.
type PostgresDB struct {
	*Database
	// BaseDSN is the disposable server DSN used for maintenance operations.
	BaseDSN string
	// DatabaseName is the random, disposable database owned by this instance.
	DatabaseName string
}

// NewPostgresDB creates an isolated disposable PostgreSQL database and
// AutoMigrates the schema onto it. Any setup, connection, migration or seed
// failure aborts the test immediately (fail-fast). It never silently falls back
// to SQLite or to a production connection.
//
// Invocation (see the constants in db.go):
//
//	TRAGGO_TEST_POSTGRES_DSN='postgres://user:pass@127.0.0.1:5432/postgres?sslmode=disable' \
//		go test ./statistics/... ./test/... -run 'Postgres|_postgres' -v
//
// Set TRAGGO_TEST_POSTGRES_REQUIRED=1 to turn a missing DSN or a failure to
// connect into a hard error instead of a skip (for CI).
func NewPostgresDB(t *testing.T, cfg PostgresConfig) *PostgresDB {
	if !cfg.Enabled() {
		if cfg.Required {
			t.Fatalf("%s is not set but %s=1: PostgreSQL tests are required and cannot run",
				PostgresDSNEnv, PostgresRequiredEnv)
		}
		t.Skipf("%s is not set; skipping PostgreSQL test. Set %s to a disposable test DSN "+
			"(and %s=1 in CI to require execution).",
			PostgresDSNEnv, PostgresDSNEnv, PostgresRequiredEnv)
	}
	pdb, err := newPostgresDB(cfg, t)
	if err != nil {
		t.Fatalf("PostgreSQL test harness setup failed (no fallback): %v", err)
	}
	return pdb
}

func newPostgresDB(cfg PostgresConfig, t *testing.T) (*PostgresDB, error) {
	dbName := fmt.Sprintf("traggo_test_%d_%d", os.Getpid(), time.Now().UnixNano())

	// 1. Maintenance connection: create the disposable database.
	admin, err := sql.Open("postgres", cfg.BaseDSN)
	if err != nil {
		return nil, fmt.Errorf("open maintenance connection: %w", err)
	}
	if err := admin.Ping(); err != nil {
		admin.Close()
		return nil, fmt.Errorf("ping disposable PostgreSQL server (check TRAGGO_TEST_POSTGRES_DSN): %w", err)
	}
	if _, err := admin.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName)); err != nil {
		admin.Close()
		return nil, fmt.Errorf("create disposable database %q: %w", dbName, err)
	}
	admin.Close()

	// 2. Connection to the new database with GORM + AutoMigrate.
	targetDSN, err := postgresDSNForDatabase(cfg.BaseDSN, dbName)
	if err != nil {
		dropPostgresDatabase(cfg.BaseDSN, dbName)
		return nil, err
	}
	// database.New runs AutoMigrate(model.All()...) itself and propagates any
	// migration error, so the harness owns no separate migration step. The
	// model.All() order is already topologically sorted (users before
	// tag_definitions etc.), so production startup migration works on a fresh
	// PostgreSQL database without a manual table-first workaround. A failed
	// connect or migration must fail the harness (fail-fast), never silently
	// proceed.
	gdb, err := database.New("postgres", targetDSN)
	if err != nil {
		dropPostgresDatabase(cfg.BaseDSN, dbName)
		return nil, fmt.Errorf("connect/AutoMigrate disposable database %q: %w", dbName, err)
	}

	return &PostgresDB{
		Database:     &Database{DB: gdb, t: t},
		BaseDSN:      cfg.BaseDSN,
		DatabaseName: dbName,
	}, nil
}

// Close closes the fixture connection and drops the disposable database. It is
// idempotent and safe to call even if setup partially failed.
func (p *PostgresDB) Close() {
	if p == nil {
		return
	}
	if p.Database != nil && p.DB != nil {
		p.DB.Close()
	}
	if p.BaseDSN != "" && p.DatabaseName != "" {
		dropPostgresDatabase(p.BaseDSN, p.DatabaseName)
		p.DatabaseName = ""
	}
}

func dropPostgresDatabase(baseDSN, dbName string) {
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return
	}
	defer admin.Close()
	// WITH (FORCE) terminates any lingering connections so cleanup is reliable.
	_, _ = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %q WITH (FORCE)", dbName))
}

// postgresDSNForDatabase rewrites a lib/pq connection string so it targets the
// given database. Both URL (postgres://) and keyword/value forms are supported.
func postgresDSNForDatabase(dsn, dbName string) (string, error) {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(trimmed, "postgres://") || strings.HasPrefix(trimmed, "postgresql://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("parse postgres URL DSN: %w", err)
		}
		u.Path = "/" + dbName
		return u.String(), nil
	}

	// keyword=value form: replace the dbname= parameter (or append it).
	fields := strings.Fields(trimmed)
	var replaced bool
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
