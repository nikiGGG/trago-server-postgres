package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"    // enable the mysql dialect
	_ "github.com/jinzhu/gorm/dialects/postgres" // enable the postgres dialect
	_ "github.com/jinzhu/gorm/dialects/sqlite"   // enable the sqlite3 dialect
	"github.com/rs/zerolog/log"
	"github.com/traggo/server/logger"
	"github.com/traggo/server/model"
)

var mkdirAll = os.MkdirAll

// migrateSchema runs the production schema migration on the freshly opened
// pool. It is a package variable so the internal database tests can observe the
// exact owned pool used during a failed initialization and inject controlled
// failures without substituting an unrelated connection. Overriding tests must
// restore it (see restoreMigrationHooks in migration_test.go).
var migrateSchema = func(db *gorm.DB) error {
	return db.AutoMigrate(model.All()...).Error
}

// closePool closes the database pool opened during initialization. Like
// migrateSchema it is a package variable so tests can count cleanup runs and
// inject a controlled close failure without replacing an unrelated connection.
var closePool = func(db *gorm.DB) error {
	return db.Close()
}

// New creates a gorm instance.
func New(dialect, connection string) (*gorm.DB, error) {
	createDirectoryIfSqlite(dialect, connection)

	db, err := gorm.Open(dialect, connection)
	if err != nil {
		return nil, err
	}
	db.LogMode(true)
	db.SetLogger(&logger.DatabaseLogger{})

	// We normally don't need that much connections, so we limit them. F.ex. mysql complains about
	// "too many connections".
	db.DB().SetMaxOpenConns(10)

	if dialect == "sqlite3" {
		// We use the database connection inside the handlers from the http
		// framework, therefore concurrent access occurs. Sqlite cannot handle
		// concurrent writes, so we limit sqlite to one connection.
		// see https://github.com/mattn/go-sqlite3/issues/274
		db.DB().SetMaxOpenConns(1)
		db.Exec("PRAGMA foreign_keys = ON")
	}

	log.Debug().Msg("Auto migrating schema's")
	if err := migrateSchema(db); err != nil {
		// The connection succeeded but the schema migration failed. A failed
		// initialization must never return a usable handle, so close the pool
		// opened by it and report the migration cause. If closing also fails,
		// the migration defect is still the real problem and must remain the
		// reported error; a cleanup failure must not hide it.
		if cerr := closePool(db); cerr != nil {
			log.Debug().Err(cerr).Msg("Failed to close database pool after migration error")
		}
		return nil, fmt.Errorf("migrate database schema: %w", err)
	}

	log.Debug().Msg("Database initialized")
	return db, nil
}

func createDirectoryIfSqlite(dialect string, connection string) {
	if dialect == "sqlite3" {
		if _, err := os.Stat(filepath.Dir(connection)); os.IsNotExist(err) {
			if err := mkdirAll(filepath.Dir(connection), 0777); err != nil {
				panic(err)
			}
		}
	}
}
