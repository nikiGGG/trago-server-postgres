package database

import (
	"errors"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreMigrationHooks restores the package-level migrateSchema and closePool
// seams after a test overrides them, so tests are order independent. These
// tests live in the internal database package to observe the exact owned pool
// and inject controlled failures without substituting an unrelated fixture
// connection. (Importing the external test harness here would form a cycle.)
func restoreMigrationHooks(t *testing.T, origMigrate, origClose func(*gorm.DB) error) {
	t.Cleanup(func() {
		migrateSchema = origMigrate
		closePool = origClose
	})
}

// TestNew_closesOwnedPoolOnMigrationError proves that a failed initialization
// closes the exact pool it opened: the retained owned pool must no longer be
// usable (Ping fails) and New must not return a usable handle.
func TestNew_closesOwnedPoolOnMigrationError(t *testing.T) {
	origMigrate, origClose := migrateSchema, closePool
	restoreMigrationHooks(t, origMigrate, origClose)

	var owned *gorm.DB
	migrateSchema = func(db *gorm.DB) error {
		owned = db // retain the exact owned pool used during the failure
		return errors.New("sentinel migration failure")
	}
	closePool = func(db *gorm.DB) error { return db.Close() }

	db, err := New("sqlite3", ":memory:")
	require.Nil(t, db, "failed initialization must not return a usable handle")
	require.Error(t, err, "migration failure must be propagated")
	require.Contains(t, err.Error(), "sentinel migration failure")
	require.NotNil(t, owned, "the owned pool must be retained for inspection")

	// The retained owned pool must be closed: Ping on it must fail. A test that
	// fails to close the pool leaks it and must fail here.
	require.Error(t, owned.DB().Ping(), "owned pool must be closed after failed initialization")
}

// TestNew_cleanupRunsOnce proves cleanup closes the owned pool exactly once on
// a migration failure, never twice and never zero times.
func TestNew_cleanupRunsOnce(t *testing.T) {
	origMigrate, origClose := migrateSchema, closePool
	restoreMigrationHooks(t, origMigrate, origClose)

	closeCalls := 0
	migrateSchema = func(db *gorm.DB) error { return errors.New("migration boom") }
	closePool = func(db *gorm.DB) error {
		closeCalls++
		return db.Close()
	}

	db, err := New("sqlite3", ":memory:")
	require.Nil(t, db)
	require.Error(t, err)
	assert.Equal(t, 1, closeCalls, "cleanup must run exactly once on failed initialization")
}

// TestNew_migrationCauseSurvivesCloseFailure proves a cleanup error cannot
// replace or hide the migration cause: the caller must still receive the
// migration error, and never a usable handle.
func TestNew_migrationCauseSurvivesCloseFailure(t *testing.T) {
	origMigrate, origClose := migrateSchema, closePool
	restoreMigrationHooks(t, origMigrate, origClose)

	migrateSchema = func(db *gorm.DB) error { return errors.New("migration boom") }
	closePool = func(db *gorm.DB) error { return errors.New("close boom") }

	db, err := New("sqlite3", ":memory:")
	require.Nil(t, db, "a cleanup error must not yield a usable handle")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "migration boom", "migration cause must be preserved")
	assert.NotContains(t, err.Error(), "close boom", "a cleanup error must not replace the migration cause")
}
