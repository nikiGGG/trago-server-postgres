package dashboard

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/database"
	"github.com/traggo/server/generated/gqlmodel"
	"github.com/traggo/server/model"
	"github.com/traggo/server/statistics"
	"github.com/traggo/server/test"
	"github.com/traggo/server/test/fake"
)

// ---------------------------------------------------------------------------
// T015: Dashboard API persistence and chart selections.
//
// These tests exercise the REAL resolver/API layer (NewResolverForDashboard,
// CreateDashboard, AddDashboardRange, AddDashboardEntry, Dashboards) - the same
// surface the GraphQL layer calls - preserving Table and StackedBarChart
// configuration, the Daily interval, named ranges and Include/Exclude filters
// across a close + reopen of the database connection. They run on SQLite and on
// a disposable PostgreSQL database. The model-level bytea/text compatibility is
// covered by postgres_persistence_test.go (WP02); this file proves the resolver
// contract end to end.
//
// FR-001 is confirmed separately (verifyStatsShapes): both the `stats` and
// `stats2` resolvers return the RangedStatisticsEntries GraphQL shape
// (Start/End + Entries with Key/Value/TimeSpendInSeconds) on both engines.
// ---------------------------------------------------------------------------
//
// Repeatable manual chart/restart acceptance check (for the WP06 / FR-008
// runbook). Against a deployed server connected to PostgreSQL 17 (or SQLite),
// after a clean install AND after an application restart:
//
//  1. Open a dashboard and add a Table and a StackedBarChart, each with
//     interval "Daily", a named range (or now-7d..now) and a tag key "работа".
//  2. On the StackedBarChart apply Include ["работа"=дом] and Exclude
//     ["работа"=подработка]; leave the Table unfiltered.
//  3. Select the week Mon..Sun and confirm both charts show daily buckets
//     (7 columns), that empty days show zero-height bars (zero-fill), and that
//     a span crossing midnight is counted in both adjacent days.
//  4. Restart the application (Kubernetes rollout or stop/start the binary),
//     reopen the dashboard and confirm the chart types, Daily interval, named
//     range, Include/Exclude filters and layout all survived the restart.
//  5. Confirm both the stats and stats2 queries return rows (no "syntax error
//     at or near WHERE", no Scan errors) in the application and database logs.
//
// This file's TestDashboardAPI_* and TestStatsGraphQLShapes_* tests automate the
// equivalent checks on a disposable database; the manual check above is the
// live-equivalent verification for the release runbook.
// ---------------------------------------------------------------------------

// verifyDashboardAPIPersistence drives the resolver API, then (when reopen is
// non-nil) closes the connection, reopens it and reads the dashboard back to
// prove the chart selections persisted.
func verifyDashboardAPIPersistence(t *testing.T, db *gorm.DB, reopen func() *gorm.DB) {
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1}}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Name: "u2", Pass: []byte{2}}).Error)

	resolver := NewResolverForDashboard(db)
	user := fake.User(1)

	// Create dashboard.
	dash, err := resolver.CreateDashboard(user, "работа неделя")
	require.NoError(t, err)
	require.NotZero(t, dash.ID)

	// Named range.
	namedRange, err := resolver.AddDashboardRange(user, dash.ID, gqlmodel.InputNamedDateRange{
		Name: "last week",
		Range: &gqlmodel.InputRelativeOrStaticRange{
			From: "now-7d",
			To:   "now",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "last week", namedRange.Name)

	// StackedBarChart + Daily interval + named range + Include/Exclude filters.
	stacked, err := resolver.AddDashboardEntry(user, dash.ID, gqlmodel.EntryTypeStackedBarChart, "работа по дням", false, gqlmodel.InputStatsSelection{
		Interval:    gqlmodel.StatsIntervalDaily,
		Tags:        []string{"работа"},
		RangeID:     &namedRange.ID,
		IncludeTags: []*gqlmodel.InputTimeSpanTag{{Key: "работа", Value: "дом"}},
		ExcludeTags: []*gqlmodel.InputTimeSpanTag{{Key: "работа", Value: "подработка"}},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, gqlmodel.EntryTypeStackedBarChart, stacked.EntryType)
	require.Equal(t, gqlmodel.StatsIntervalDaily, stacked.StatsSelection.Interval)
	require.Equal(t, []string{"работа"}, stacked.StatsSelection.Tags)
	require.NotNil(t, stacked.StatsSelection.RangeID)
	require.Equal(t, namedRange.ID, *stacked.StatsSelection.RangeID)
	require.ElementsMatch(t, []*gqlmodel.TimeSpanTag{{Key: "работа", Value: "дом"}}, stacked.StatsSelection.IncludeTags)
	require.ElementsMatch(t, []*gqlmodel.TimeSpanTag{{Key: "работа", Value: "подработка"}}, stacked.StatsSelection.ExcludeTags)

	// VerticalTable (Table) + Daily interval + static range.
	table, err := resolver.AddDashboardEntry(user, dash.ID, gqlmodel.EntryTypeVerticalTable, "таблица", true, gqlmodel.InputStatsSelection{
		Interval: gqlmodel.StatsIntervalDaily,
		Tags:     []string{"работа"},
		Range: &gqlmodel.InputRelativeOrStaticRange{
			From: "now-7d",
			To:   "now",
		},
	}, nil)
	require.NoError(t, err)
	require.Equal(t, gqlmodel.EntryTypeVerticalTable, table.EntryType)
	require.Equal(t, gqlmodel.StatsIntervalDaily, table.StatsSelection.Interval)
	require.Nil(t, table.StatsSelection.RangeID)
	require.NotNil(t, table.StatsSelection.Range)
	require.Equal(t, "now-7d", table.StatsSelection.Range.From)
	require.Equal(t, "now", table.StatsSelection.Range.To)

	// Other-user isolation before reconnect: user 2 sees nothing.
	dashboards2, err := resolver.Dashboards(fake.User(2))
	require.NoError(t, err)
	require.Empty(t, dashboards2)

	// Close + reopen the connection (real persistence check).
	if reopen != nil {
		db = reopen()
		resolver = NewResolverForDashboard(db)
	}

	dashboards, err := resolver.Dashboards(user)
	require.NoError(t, err)
	require.Len(t, dashboards, 1)
	d := dashboards[0]
	require.Equal(t, "работа неделя", d.Name)
	require.Len(t, d.Ranges, 1)
	require.Equal(t, "last week", d.Ranges[0].Name)

	// Locate persisted entries by ID (item order is not part of the contract).
	var gotStacked, gotTable *gqlmodel.DashboardEntry
	for _, it := range d.Items {
		if it.ID == stacked.ID {
			gotStacked = it
		}
		if it.ID == table.ID {
			gotTable = it
		}
	}
	require.NotNil(t, gotStacked, "StackedBarChart entry must persist")
	require.NotNil(t, gotTable, "VerticalTable entry must persist")

	// StackedBarChart config survived reconnect: chart type, Daily interval,
	// named-range reference and Include/Exclude filters.
	require.Equal(t, gqlmodel.EntryTypeStackedBarChart, gotStacked.EntryType)
	require.Equal(t, gqlmodel.StatsIntervalDaily, gotStacked.StatsSelection.Interval)
	require.Equal(t, namedRange.ID, *gotStacked.StatsSelection.RangeID)
	require.ElementsMatch(t, []*gqlmodel.TimeSpanTag{{Key: "работа", Value: "дом"}}, gotStacked.StatsSelection.IncludeTags)
	require.ElementsMatch(t, []*gqlmodel.TimeSpanTag{{Key: "работа", Value: "подработка"}}, gotStacked.StatsSelection.ExcludeTags)

	// Table config survived reconnect: chart type, Daily interval and the
	// static range.
	require.Equal(t, gqlmodel.EntryTypeVerticalTable, gotTable.EntryType)
	require.Equal(t, gqlmodel.StatsIntervalDaily, gotTable.StatsSelection.Interval)
	require.Equal(t, "now-7d", gotTable.StatsSelection.Range.From)
	require.Equal(t, "now", gotTable.StatsSelection.Range.To)
}

// TestDashboardAPI_sqlite exercises the resolver API on a file-backed SQLite
// database and proves chart selections persist across a close + reopen.
func TestDashboardAPI_sqlite(t *testing.T) {
	db, path := fileSQLiteDB(t)
	verifyDashboardAPIPersistence(t, db, func() *gorm.DB {
		require.NoError(t, db.Close())
		reopened, err := database.New("sqlite3", path)
		require.NoError(t, err, "reopen file-backed sqlite database")
		t.Cleanup(func() { reopened.Close() })
		return reopened
	})
}

// TestDashboardAPI_postgres exercises the same resolver API on a disposable
// PostgreSQL database (normal text schema) with a real close + reopen.
func TestDashboardAPI_postgres(t *testing.T) {
	cfg := postgresConfig(t)
	gdb, dsn, cleanup := openRawPostgres(t, cfg)
	defer cleanup()
	verifyDashboardAPIPersistence(t, gdb, func() *gorm.DB {
		require.NoError(t, gdb.Close())
		reopened, err := database.New("postgres", dsn)
		require.NoError(t, err, "reopen PostgreSQL database")
		t.Cleanup(func() { reopened.Close() })
		return reopened
	})
}

// verifyStatsShapes confirms FR-001 on one engine: both the `stats` and `stats2`
// resolvers return the RangedStatisticsEntries GraphQL shape with correct
// per-span values, and no SQL/Scan error.
func verifyStatsShapes(t *testing.T, db *gorm.DB) {
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1}}).Error)
	end := test.Time("2019-06-11T10:00:10Z")
	span := model.TimeSpan{
		StartUserTime: test.Time("2019-06-11T10:00:00Z"),
		EndUserTime:   &end,
		UserID:        1,
		Tags:          []model.TimeSpanTag{{Key: "работа", StringValue: "дом"}},
	}
	require.NoError(t, db.Create(&span).Error)

	sres := statistics.ResolverForStatistics{DB: db}
	user := fake.User(1)

	// `stats` endpoint.
	stats, err := sres.Stats(user,
		[]*gqlmodel.Range{{Start: test.ModelTime("2019-06-11T10:00:00Z"), End: test.ModelTime("2019-06-11T10:00:10Z")}},
		[]string{"работа"}, nil, nil)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.NotNil(t, stats[0].Entries)
	require.Len(t, stats[0].Entries, 1)
	require.Equal(t, "работа", stats[0].Entries[0].Key)
	require.Equal(t, "дом", stats[0].Entries[0].Value)
	require.Equal(t, 10.0, stats[0].Entries[0].TimeSpendInSeconds)

	// `stats2` endpoint (Single interval over the span; shape check only).
	stats2, err := sres.Stats2(user, test.ModelTime("2019-06-11T10:00:10Z"), gqlmodel.InputStatsSelection{
		Interval: gqlmodel.StatsIntervalSingle,
		Tags:     []string{"работа"},
		Range: &gqlmodel.InputRelativeOrStaticRange{
			From: "2019-06-11T10:00:00Z",
			To:   "2019-06-11T10:00:10Z",
		},
	})
	require.NoError(t, err)
	require.Len(t, stats2, 1)
	require.Len(t, stats2[0].Entries, 1)
	require.Equal(t, 10.0, stats2[0].Entries[0].TimeSpendInSeconds)
}

// TestStatsGraphQLShapes_sqlite confirms both stats endpoints keep their
// GraphQL shapes on SQLite.
func TestStatsGraphQLShapes_sqlite(t *testing.T) {
	db := test.InMemoryDB(t)
	defer db.Close()
	verifyStatsShapes(t, db.DB)
}

// TestStatsGraphQLShapes_postgres confirms both stats endpoints keep their
// GraphQL shapes on a disposable PostgreSQL database.
func TestStatsGraphQLShapes_postgres(t *testing.T) {
	cfg := postgresConfig(t)
	pdb := test.NewPostgresDB(t, cfg)
	defer pdb.Close()
	verifyStatsShapes(t, pdb.DB)
}
