package dashboard

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // register the "postgres" driver
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/database"
	"github.com/traggo/server/model"
	"github.com/traggo/server/test"
)

// postgresConfig skips (or fails, when required) when no disposable PostgreSQL
// DSN is configured, mirroring the WP01 harness contract.
func postgresConfig(t *testing.T) test.PostgresConfig {
	cfg := test.PostgresConfigFromEnv()
	if !cfg.Enabled() {
		if cfg.Required {
			t.Fatalf("%s not set but %s=1; PostgreSQL tests are required", test.PostgresDSNEnv, test.PostgresRequiredEnv)
		}
		t.Skipf("%s not set; skipping PostgreSQL persistence test", test.PostgresDSNEnv)
	}
	return cfg
}

// postgresDSNForDatabase rewrites a lib/pq URL DSN to target a specific database.
func postgresDSNForDatabase(baseDSN, dbName string) string {
	u, err := url.Parse(baseDSN)
	if err != nil {
		return baseDSN
	}
	u.Path = "/" + dbName
	return u.String()
}

// openRawPostgres creates a brand-new disposable PostgreSQL database and opens it
// through the real production startup path database.New (which runs
// AutoMigrate(model.All()...)). This proves that startup migration itself works on
// a fresh schema. It returns the connection, the target DSN, and a cleanup func
// that closes the connection and drops the database.
func openRawPostgres(t *testing.T, cfg test.PostgresConfig) (*gorm.DB, string, func()) {
	dbName := fmt.Sprintf("traggo_wp02_%d_%d", os.Getpid(), time.Now().UnixNano())

	admin, err := sql.Open("postgres", cfg.BaseDSN)
	require.NoError(t, err, "open maintenance connection")
	require.NoError(t, admin.Ping(), "ping disposable PostgreSQL server")
	_, err = admin.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName))
	require.NoError(t, admin.Close())
	require.NoError(t, err, "create disposable database")

	dsn := postgresDSNForDatabase(cfg.BaseDSN, dbName)
	gdb, err := database.New("postgres", dsn)
	if err != nil {
		dropPostgresDatabase(cfg.BaseDSN, dbName)
		t.Fatalf("production startup AutoMigrate failed on a fresh PostgreSQL schema: %v", err)
	}

	cleanup := func() {
		gdb.Close()
		dropPostgresDatabase(cfg.BaseDSN, dbName)
	}
	return gdb, dsn, cleanup
}

func dropPostgresDatabase(baseDSN, dbName string) {
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return
	}
	defer admin.Close()
	_, _ = admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %q WITH (FORCE)", dbName))
}

// pgColumnType returns the PostgreSQL data_type of a column.
func pgColumnType(t *testing.T, db *gorm.DB, table, column string) string {
	var typ string
	row := db.Raw(
		"SELECT data_type FROM information_schema.columns WHERE table_name = ? AND column_name = ?",
		table, column,
	).Row()
	require.NoError(t, row.Scan(&typ))
	return typ
}

func seedUserAndDashboard(t *testing.T, db *gorm.DB) {
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1}}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Name: "u2", Pass: []byte{2}}).Error)
	require.NoError(t, db.Create(&model.Dashboard{ID: 1, UserID: 1, Name: "dash-1"}).Error)
	require.NoError(t, db.Create(&model.Dashboard{ID: 2, UserID: 2, Name: "dash-2"}).Error)
}

// ============================================================
// T006: compatibility with PostgreSQL text AND legacy bytea columns
// ============================================================

// TestDashboardEntryPersistence_text proves string enum values round-trip through
// normal PostgreSQL text columns (the clean schema).
func TestDashboardEntryPersistence_text(t *testing.T) {
	cfg := postgresConfig(t)
	pdb := test.NewPostgresDB(t, cfg)
	defer pdb.Close()

	seedUserAndDashboard(t, pdb.DB)

	entry := model.DashboardEntry{
		DashboardID:     1,
		Title:           "my chart",
		Total:           true,
		Type:            model.TypeStackedBarChart,
		Interval:        model.IntervalWeekly,
		Keys:            "работа,дом",
		RangeID:         model.NoRangeIDDefined,
		RangeFrom:       "now-30d",
		RangeTo:         "now",
		MobilePosition:  `{"x":0,"y":0,"w":1,"h":2,"minW":1,"minH":2}`,
		DesktopPosition: `{"x":1,"y":2,"w":3,"h":4,"minW":1,"minH":2}`,
		TagFilters: []model.DashboardTagFilter{
			{Key: "работа", StringValue: "дом", Include: true},
			{Key: "работа", StringValue: "офис", Include: false},
		},
	}
	require.NoError(t, pdb.Save(&entry).Error, "insert dashboard entry on text schema")
	require.NotZero(t, entry.ID)

	// read back
	var got model.DashboardEntry
	require.NoError(t, pdb.Preload("TagFilters").First(&got, entry.ID).Error)
	require.Equal(t, model.TypeStackedBarChart, got.Type)
	require.Equal(t, model.IntervalWeekly, got.Interval)
	require.Equal(t, "now-30d", got.RangeFrom)
	require.Equal(t, "now", got.RangeTo)
	require.Equal(t, "работа,дом", got.Keys)
	require.Len(t, got.TagFilters, 2)

	// update type/interval
	require.NoError(t, pdb.Model(&got).Update("type", model.TypeLineChart).Error)
	require.NoError(t, pdb.Model(&got).Update("interval", model.IntervalDaily).Error)

	var got2 model.DashboardEntry
	require.NoError(t, pdb.First(&got2, entry.ID).Error)
	require.Equal(t, model.TypeLineChart, got2.Type)
	require.Equal(t, model.IntervalDaily, got2.Interval)
}

// TestDashboardEntryPersistence_bytea proves the same string enum values round-trip
// through the legacy bytea columns produced by the task-114 workaround. If binding
// to bytea failed, this test fails and would require a binding-boundary fix
// (never a destructive schema conversion).
func TestDashboardEntryPersistence_bytea(t *testing.T) {
	cfg := postgresConfig(t)
	pdb := test.NewPostgresDB(t, cfg)
	defer pdb.Close()

	seedUserAndDashboard(t, pdb.DB)

	// Simulate the deployed workaround: enum columns become bytea.
	for _, stmt := range []string{
		`ALTER TABLE dashboard_entries ALTER COLUMN type TYPE bytea USING type::bytea`,
		`ALTER TABLE dashboard_entries ALTER COLUMN interval TYPE bytea USING interval::bytea`,
	} {
		require.NoError(t, pdb.Exec(stmt).Error, "convert column to bytea")
	}
	require.Equal(t, "bytea", pgColumnType(t, pdb.DB, "dashboard_entries", "type"))
	require.Equal(t, "bytea", pgColumnType(t, pdb.DB, "dashboard_entries", "interval"))

	// insert string values into bytea columns
	entry := model.DashboardEntry{
		ID:          1,
		DashboardID: 1,
		Title:       "bytea chart",
		Type:        model.TypeBarChart,
		Interval:    model.IntervalHourly,
		Keys:        "k",
		RangeID:     model.NoRangeIDDefined,
	}
	require.NoError(t, pdb.Save(&entry).Error, "insert dashboard entry into bytea columns")

	// read back
	var got model.DashboardEntry
	require.NoError(t, pdb.First(&got, 1).Error)
	require.Equal(t, model.TypeBarChart, got.Type)
	require.Equal(t, model.IntervalHourly, got.Interval)

	// update
	require.NoError(t, pdb.Model(&got).Update("type", model.TypePieChart).Error)
	require.NoError(t, pdb.Model(&got).Update("interval", model.IntervalMonthly).Error)

	var got2 model.DashboardEntry
	require.NoError(t, pdb.First(&got2, 1).Error)
	require.Equal(t, model.TypePieChart, got2.Type)
	require.Equal(t, model.IntervalMonthly, got2.Interval)
}

// ============================================================
// T007: lifecycle persistence (file-backed SQLite + PG reconnect)
// ============================================================

// fileSQLiteDB creates a fresh file-backed SQLite database (auto-migrated) and
// returns it together with its path so a test can close and reopen it.
func fileSQLiteDB(t *testing.T) (*gorm.DB, string) {
	path := filepath.Join(t.TempDir(), "traggo.db")
	db, err := database.New("sqlite3", path)
	require.NoError(t, err, "create file-backed sqlite database")
	return db, path
}

// TestDashboardLifecycle_fileSQLite exercises create/edit/reopen of dashboard
// entries, chart types, intervals, named ranges, filters and layout fields on a
// file-backed SQLite database, then verifies persistence across a close/reopen
// of the connection plus other-user isolation.
func TestDashboardLifecycle_fileSQLite(t *testing.T) {
	db, path := fileSQLiteDB(t)

	seedUserAndDashboard(t, db)

	// named range + a fully-featured entry
	dashRange := model.DashboardRange{
		DashboardID: 1,
		Name:        "last month",
		Editable:    true,
		From:        "now-30d",
		To:          "now",
	}
	require.NoError(t, db.Save(&dashRange).Error)

	entry := model.DashboardEntry{
		DashboardID:     1,
		Title:           "stacked",
		Total:           false,
		Type:            model.TypeStackedBarChart,
		Interval:        model.IntervalDaily,
		Keys:            "работа",
		RangeID:         dashRange.ID,
		MobilePosition:  `{"x":0,"y":0,"w":2,"h":1,"minW":2,"minH":1}`,
		DesktopPosition: `{"x":0,"y":0,"w":4,"h":2,"minW":2,"minH":2}`,
		TagFilters: []model.DashboardTagFilter{
			{Key: "работа", StringValue: "дом", Include: true},
			{Key: "работа", StringValue: "офис", Include: false},
			{Key: "работа", StringValue: "подработка", Include: true},
		},
	}
	require.NoError(t, db.Save(&entry).Error)

	// edit the entry: change chart type + interval (clear the already-saved
	// association so GORM does not re-save it during Update)
	entry.TagFilters = nil
	require.NoError(t, db.Model(&entry).Update("type", model.TypeBarChart).Error)
	require.NoError(t, db.Model(&entry).Update("interval", model.IntervalWeekly).Error)

	// a dashboard owned by user 2 with no entries (for isolation check)
	require.NoError(t, db.Save(&model.DashboardRange{DashboardID: 2, Name: "r2", From: "now-1d", To: "now"}).Error)

	// close and reopen the database connection from disk
	require.NoError(t, db.Close())
	db2, err := database.New("sqlite3", path)
	require.NoError(t, err, "reopen file-backed sqlite database")
	defer db2.Close()

	// user 1's dashboard and entry persisted with edited fields
	var d1 model.Dashboard
	require.NoError(t, db2.First(&d1, 1).Error)
	require.Equal(t, "dash-1", d1.Name)

	var gotEntry model.DashboardEntry
	require.NoError(t, db2.Preload("TagFilters").First(&gotEntry, entry.ID).Error)
	require.Equal(t, model.TypeBarChart, gotEntry.Type)
	require.Equal(t, model.IntervalWeekly, gotEntry.Interval)
	require.Equal(t, dashRange.ID, gotEntry.RangeID)
	require.Equal(t, `{"x":0,"y":0,"w":4,"h":2,"minW":2,"minH":2}`, gotEntry.DesktopPosition)
	require.Len(t, gotEntry.TagFilters, 3)
	require.ElementsMatch(t, []string{"дом", "офис", "подработка"}, tagFilterValues(gotEntry.TagFilters))

	// named range persisted
	var gotRange model.DashboardRange
	require.NoError(t, db2.First(&gotRange, dashRange.ID).Error)
	require.Equal(t, "last month", gotRange.Name)

	// other-user isolation after reconnect: user 2 owns only dashboard 2
	var countUser1 int
	require.NoError(t, db2.Model(&model.Dashboard{}).Where("user_id = ?", 1).Count(&countUser1).Error)
	require.Equal(t, 1, countUser1)
	var countUser2 int
	require.NoError(t, db2.Model(&model.Dashboard{}).Where("user_id = ?", 2).Count(&countUser2).Error)
	require.Equal(t, 1, countUser2)

	// user 2 cannot read user 1's entry via a user-scoped query
	var leak int64
	require.NoError(t, db2.Model(&model.DashboardEntry{}).
		Joins("JOIN dashboards ON dashboards.id = dashboard_entries.dashboard_id").
		Where("dashboards.user_id = ? AND dashboard_entries.id = ?", 2, gotEntry.ID).
		Count(&leak).Error)
	require.Equal(t, int64(0), leak, "user 2 must not see user 1's dashboard entry")
}

// tagFilterValues returns the distinct StringValues of the tag filters.
func tagFilterValues(filters []model.DashboardTagFilter) []string {
	out := make([]string, 0, len(filters))
	for _, f := range filters {
		out = append(out, f.StringValue)
	}
	return out
}

// TestDashboardLifecycle_pgReconnect proves dashboard entries survive a real
// close + reopen of the PostgreSQL connection via database.New.
func TestDashboardLifecycle_pgReconnect(t *testing.T) {
	cfg := postgresConfig(t)
	gdb, dsn, cleanup := openRawPostgres(t, cfg)
	seedUserAndDashboard(t, gdb)

	entry := model.DashboardEntry{
		ID:          1,
		DashboardID: 1,
		Title:       "persistent",
		Type:        model.VerticalTable,
		Interval:    model.IntervalYearly,
		Keys:        "k",
		RangeID:     model.NoRangeIDDefined,
	}
	require.NoError(t, gdb.Save(&entry).Error)

	// close and reopen via the production path
	require.NoError(t, gdb.Close())
	gdb2, err := database.New("postgres", dsn)
	require.NoError(t, err, "reopen PostgreSQL database")
	defer gdb2.Close()
	defer cleanup()

	var got model.DashboardEntry
	require.NoError(t, gdb2.First(&got, 1).Error)
	require.Equal(t, model.VerticalTable, got.Type)
	require.Equal(t, model.IntervalYearly, got.Interval)
}

// ============================================================
// T008: startup migration on existing schemas (text + bytea)
// ============================================================

// TestStartupMigration_reopen_textAndBytea initialises both PostgreSQL schema
// variants with database.New, seeds values, converts to the legacy bytea variant,
// then closes and reopens via database.New. It asserts existing values/types are
// retained and that AutoMigrate does not destructively alter the bytea columns.
func TestStartupMigration_reopen_textAndBytea(t *testing.T) {
	cfg := postgresConfig(t)

	run := func(t *testing.T, convertToBytea bool) {
		gdb, dsn, cleanup := openRawPostgres(t, cfg)
		defer cleanup()

		seedUserAndDashboard(t, gdb)
		entry := model.DashboardEntry{
			ID:          1,
			DashboardID: 1,
			Title:       "startup",
			Type:        model.TypeLineChart,
			Interval:    model.IntervalDaily,
			Keys:        "k",
			RangeID:     model.NoRangeIDDefined,
		}
		require.NoError(t, gdb.Save(&entry).Error)
		dev := model.Device{
			ID:     1,
			Token:  "tok-1",
			Name:   "dev-1",
			UserID: 1,
			Type:   model.TypeNoExpiry,
		}
		require.NoError(t, gdb.Create(&dev).Error)

		if convertToBytea {
			for _, stmt := range []string{
				`ALTER TABLE dashboard_entries ALTER COLUMN type TYPE bytea USING type::bytea`,
				`ALTER TABLE dashboard_entries ALTER COLUMN interval TYPE bytea USING interval::bytea`,
				`ALTER TABLE devices ALTER COLUMN type TYPE bytea USING type::bytea`,
			} {
				require.NoError(t, gdb.Exec(stmt).Error, "convert enum columns to bytea")
			}
		}

		// Record column types before reopen.
		typeBefore := pgColumnType(t, gdb, "dashboard_entries", "type")
		require.NoError(t, gdb.Close())

		// Reopen through the production startup path.
		gdb2, err := database.New("postgres", dsn)
		require.NoError(t, err, "reopen existing schema via database.New")
		defer gdb2.Close()

		// Column type must be retained (no destructive text/bytea conversion).
		typeAfter := pgColumnType(t, gdb2, "dashboard_entries", "type")
		require.Equal(t, typeBefore, typeAfter, "startup migration must not change column type")

		// Existing values are retained and readable.
		var gotEntry model.DashboardEntry
		require.NoError(t, gdb2.First(&gotEntry, 1).Error)
		require.Equal(t, model.TypeLineChart, gotEntry.Type)
		require.Equal(t, model.IntervalDaily, gotEntry.Interval)

		var gotDev model.Device
		require.NoError(t, gdb2.First(&gotDev, 1).Error)
		require.Equal(t, model.TypeNoExpiry, gotDev.Type)
	}

	t.Run("text_schema", func(t *testing.T) { run(t, false) })
	t.Run("bytea_schema", func(t *testing.T) { run(t, true) })
}
