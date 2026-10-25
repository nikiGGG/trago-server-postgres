package test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/traggo/server/model"
)

// The following two tests exercise the harness without any PostgreSQL server, so
// they always run. They prove the DSN-rewrite and config parsing logic used by
// every disposable PostgreSQL test.

func TestPostgresDSNForDatabase(t *testing.T) {
	urlDSN, err := postgresDSNForDatabase("postgres://user:pass@h:5432/postgres?sslmode=disable", "traggo_test_x")
	require.NoError(t, err)
	require.Contains(t, urlDSN, "/traggo_test_x")
	require.NotContains(t, urlDSN, "/postgres")

	kvDSN, err := postgresDSNForDatabase("host=h dbname=postgres user=u", "traggo_test_x")
	require.NoError(t, err)
	require.Contains(t, kvDSN, "dbname=traggo_test_x")
	require.NotContains(t, kvDSN, "dbname=postgres")

	appendKV, err := postgresDSNForDatabase("host=h user=u", "traggo_test_x")
	require.NoError(t, err)
	require.Contains(t, appendKV, "dbname=traggo_test_x")
}

func TestPostgresConfigFromEnv(t *testing.T) {
	t.Setenv(PostgresDSNEnv, "")
	t.Setenv(PostgresRequiredEnv, "1")
	cfg := PostgresConfigFromEnv()
	require.False(t, cfg.Enabled())
	require.True(t, cfg.Required)

	t.Setenv(PostgresDSNEnv, "postgres://u:p@h:5432/postgres")
	t.Setenv(PostgresRequiredEnv, "0")
	cfg = PostgresConfigFromEnv()
	require.True(t, cfg.Enabled())
	require.False(t, cfg.Required)
}

// ---- The following tests require a disposable PostgreSQL server (opt-in). ----

// skipOrFailPostgres skips when no DSN is configured, or fails when required.
func skipOrFailPostgres(t *testing.T) PostgresConfig {
	cfg := PostgresConfigFromEnv()
	if !cfg.Enabled() {
		if cfg.Required {
			t.Fatalf("%s not set but %s=1; PostgreSQL tests are required",
				PostgresDSNEnv, PostgresRequiredEnv)
		}
		t.Skipf("%s not set; skipping PostgreSQL harness test", PostgresDSNEnv)
	}
	return cfg
}

// TestPostgresDB_invalidConnection verifies fail-fast on an invalid test
// connection: setup must return an error, never a silently-empty harness.
func TestPostgresDB_invalidConnection(t *testing.T) {
	cfg := PostgresConfig{
		// Port 1 is effectively never a live PostgreSQL server.
		BaseDSN: "postgres://postgres:postgres@127.0.0.1:1/postgres?sslmode=disable&connect_timeout=2",
	}
	_, err := newPostgresDB(cfg, t)
	require.Error(t, err, "invalid test connection must fail fast, not return a usable harness")
}

// TestPostgresDB_isolation verifies that two disposable databases are fully
// isolated: data written into one is invisible to the other.
func TestPostgresDB_isolation(t *testing.T) {
	cfg := skipOrFailPostgres(t)

	a := NewPostgresDB(t, cfg)
	defer a.Close()
	b := NewPostgresDB(t, cfg)
	defer b.Close()

	require.NotEqual(t, a.DatabaseName, b.DatabaseName, "each test must get a distinct database")

	require.NoError(t, a.Create(&model.User{ID: 1, Name: "iso", Pass: []byte{1}}).Error)

	var countA, countB int
	require.NoError(t, a.Model(&model.User{}).Count(&countA).Error)
	require.NoError(t, b.Model(&model.User{}).Count(&countB).Error)
	require.Equal(t, 1, countA, "fixture data present in its own database")
	require.Equal(t, 0, countB, "fixture data must not leak into another test's database")
}

// TestPostgresDB_cleanup verifies that Close() drops the disposable database so
// repeated runs never accumulate databases on the test server.
func TestPostgresDB_cleanup(t *testing.T) {
	cfg := skipOrFailPostgres(t)

	p := NewPostgresDB(t, cfg)
	name := p.DatabaseName
	p.Close()

	admin, err := sql.Open("postgres", cfg.BaseDSN)
	require.NoError(t, err)
	defer admin.Close()
	var exists int
	require.NoError(t, admin.QueryRow("SELECT count(*) FROM pg_database WHERE datname = $1", name).Scan(&exists))
	require.Equal(t, 0, exists, "disposable database must be dropped on Close")
}
