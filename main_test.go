package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/config"
	"github.com/traggo/server/logger"
)

// helperProcessEnv marks a test binary invocation as the migration-failure
// helper subprocess (see TestInitDatabase_fatalOnMigrationFailure).
const helperProcessEnv = "TRAGGO_MIGRATION_FAIL_HELPER"

// migrationFailureDSNEnv carries the migration-conflict database path to the
// helper subprocess.
const migrationFailureDSNEnv = "TRAGGO_MIGRATION_FAIL_DSN"

// TestInitDatabase_fatalOnMigrationFailure exercises the production startup
// fatal boundary in a subprocess. zerolog Fatal terminates the process, so the
// helper runs in its own process (same test binary, filtered by -test.run) to
// prove that a real migration failure exits nonzero with the migration error
// and never proceeds as a successful startup.
func TestInitDatabase_fatalOnMigrationFailure(t *testing.T) {
	if os.Getenv(helperProcessEnv) == "1" {
		runMigrationFailureHelper()
		return
	}

	// Parent: create a real migration-conflict SQLite database (a view named
	// users blocks CREATE TABLE users during AutoMigrate).
	path := filepath.Join(t.TempDir(), "conflict.db")
	sqlDB, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = sqlDB.Exec(`CREATE VIEW users AS SELECT 1 AS id`)
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	cmd := exec.Command(os.Args[0], "-test.run=^TestInitDatabase_fatalOnMigrationFailure$")
	cmd.Env = append(os.Environ(), helperProcessEnv+"=1", migrationFailureDSNEnv+"="+path)
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "startup subprocess must exit nonzero on migration failure")
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "expected an ExitError from the helper subprocess")
	assert.NotEqual(t, 0, exitErr.ExitCode(), "migration failure must terminate startup with nonzero exit")

	output := string(out)
	// The subprocess must report the fatal startup boundary with the migration
	// error context, and must not have proceeded to success (no default user,
	// no "Database initialized").
	assert.Contains(t, output, "Failed to connect to the database", "fatal boundary message must be emitted")
	assert.Contains(t, output, "migrate database schema", "migration error context must be present")
	assert.Contains(t, output, "already exists", "schema-conflict cause must be present")
	assert.NotContains(t, output, "Creating default user", "startup must not create a default user after migration failure")
	assert.NotContains(t, output, "Database initialized", "startup must not report success after migration failure")
}

// runMigrationFailureHelper is the child side of
// TestInitDatabase_fatalOnMigrationFailure. It initializes logging and drives
// initDatabase against the migration-conflict database. initDatabase calls
// log.Fatal on the migration error, which terminates this process; if it ever
// returns, the test fails.
func runMigrationFailureHelper() {
	logger.Init(zerolog.WarnLevel)
	conf := config.Config{
		DatabaseDialect:    "sqlite3",
		DatabaseConnection: os.Getenv(migrationFailureDSNEnv),
		DefaultUserName:    "admin",
		DefaultUserPass:    "admin",
		PassStrength:       10,
	}
	db := initDatabase(conf)
	_ = db
	os.Exit(2) // unreachable if migration fails as expected; guards against silent success
}
