package statistics

import (
	"sort"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/generated/gqlmodel"
	"github.com/traggo/server/model"
	"github.com/traggo/server/test"
	"github.com/traggo/server/test/fake"
)

// ---------------------------------------------------------------------------
// T013: Deterministic Cyrillic weekly fixture.
//
// The fixture seeds work tags `работа=дом`, `работа=офис` and
// `работа=подработка` across a FIXED week (Monday 2019-06-10 .. Sunday
// 2019-06-16). All data is synthetic; no private production records are used.
// The week contains two fully empty days (Wednesday and Sunday) and one
// completed span that crosses Friday midnight
// (2019-06-14T23:30:00Z -> 2019-06-15T00:30:00Z).
//
// Every expected value below is an INDEPENDENT hand calculation from the
// clipped source intervals, NOT derived from the SQL implementation. They are
// asserted as fixed seconds on both engines.
//
// The user's week starts on Monday (explicit UserSetting). Stats2 is invoked
// with a fixed `now` of 2019-06-17T00:00:00Z (Monday 00:00 of the week after the
// fixture week), Interval Daily and the relative range now-7d..now, so it
// produces exactly one range per day (Monday 2019-06-10 .. Sunday 2019-06-16)
// with Stats2's existing SECOND-INCLUSIVE range builder: each day is
// [00:00:00, 23:59:59].
//
// Midnight crossing and the second-inclusive builder:
//   - The подработка span 2019-06-14T23:30:00Z -> 2019-06-15T00:30:00Z crosses
//     midnight. Because Friday's daily range ends at 23:59:59 (second-inclusive),
//     Friday's clip is 23:30:00 -> 23:59:59 = 1799s, and Saturday's clip is
//     00:00:00 -> 00:30:00 = 1800s. This 1799/1800 split is the real, preserved
//     outcome of Stats2's second-inclusive builder.
//   - This is deliberately DISTINCT from manually supplied adjacent half-open
//     midnight bounds ([Fri 00:00, Sat 00:00) and [Sat 00:00, Sun 00:00)) which
//     produce 1800s on each side; see TestMidnightCrossing_manualHalfOpen.
//     The two outcomes are asserted separately so the second-inclusive builder
//     is not silently changed.
// ---------------------------------------------------------------------------

// weekNow is the fixed "now" (Monday 00:00 of the week after the fixture week).
// With the relative range now-7d..now this yields the 7 daily ranges
// Mon 2019-06-10 .. Sun 2019-06-16, each [00:00:00, 23:59:59].
const weekNow = "2019-06-17T00:00:00Z"

// weekRange is the relative range driving the daily ranges.
var weekRange = &gqlmodel.InputRelativeOrStaticRange{From: "now-7d", To: "now"}

// weekDays lists the seven days of the fixture week in order (Mon..Sun).
var weekDays = []string{
	"2019-06-10", // Mon
	"2019-06-11", // Tue
	"2019-06-12", // Wed (empty)
	"2019-06-13", // Thu
	"2019-06-14", // Fri
	"2019-06-15", // Sat
	"2019-06-16", // Sun (empty)
}

// cyrillicSeries is the canonical output order of the three работа series.
var cyrillicSeries = []string{"дом", "офис", "подработка"}

// cyrillicDailySecs holds the hand-computed per-day seconds for each работа
// series, clipped to the second-inclusive daily ranges (see header comment).
var cyrillicDailySecs = map[string]map[string]float64{
	"2019-06-10": {"дом": 7200, "офис": 14400, "подработка": 0},
	"2019-06-11": {"дом": 5400, "офис": 0, "подработка": 10800},
	"2019-06-12": {"дом": 0, "офис": 0, "подработка": 0}, // empty day, zero-filled
	"2019-06-13": {"дом": 0, "офис": 18000, "подработка": 7200},
	"2019-06-14": {"дом": 10800, "офис": 0, "подработка": 1799}, // midnight span -> 1799s (23:30-23:59:59)
	"2019-06-15": {"дом": 0, "офис": 0, "подработка": 16200},    // 14400 (10:00-14:00) + 1800 (00:00-00:30)
	"2019-06-16": {"дом": 0, "офис": 0, "подработка": 0},        // empty day, zero-filled
}

// cyrillicWeeklyTotals are the hand-computed series totals for the whole week.
var cyrillicWeeklyTotals = map[string]float64{
	"дом":        7200 + 5400 + 10800,         // 23400 (Mon + Tue + Fri)
	"офис":       14400 + 18000,               // 32400 (Mon + Thu)
	"подработка": 10800 + 7200 + 1799 + 16200, // 35999 (Tue + Thu + Fri-midnight + Sat)
}

// cyrillicSpans returns the synthetic completed time spans for user 1.
func cyrillicSpans() []model.TimeSpan {
	return []model.TimeSpan{
		// Monday: работа=дом 09:00-11:00 (7200s)
		completedSpan("2019-06-10T09:00:00Z", "2019-06-10T11:00:00Z", tag("работа", "дом")),
		// Monday: работа=офис 11:00-15:00 (14400s)
		completedSpan("2019-06-10T11:00:00Z", "2019-06-10T15:00:00Z", tag("работа", "офис")),
		// Tuesday: работа=дом 10:00-11:30 (5400s)
		completedSpan("2019-06-11T10:00:00Z", "2019-06-11T11:30:00Z", tag("работа", "дом")),
		// Tuesday: работа=подработка 14:00-17:00 (10800s)
		completedSpan("2019-06-11T14:00:00Z", "2019-06-11T17:00:00Z", tag("работа", "подработка")),
		// Thursday: работа=офис 09:00-14:00 (18000s)
		completedSpan("2019-06-13T09:00:00Z", "2019-06-13T14:00:00Z", tag("работа", "офис")),
		// Thursday: работа=подработка 16:00-18:00 (7200s)
		completedSpan("2019-06-13T16:00:00Z", "2019-06-13T18:00:00Z", tag("работа", "подработка")),
		// Friday: работа=дом 08:00-11:00 (10800s)
		completedSpan("2019-06-14T08:00:00Z", "2019-06-14T11:00:00Z", tag("работа", "дом")),
		// Midnight crossing: работа=подработка 2019-06-14T23:30:00Z -> 2019-06-15T00:30:00Z
		completedSpan("2019-06-14T23:30:00Z", "2019-06-15T00:30:00Z", tag("работа", "подработка")),
		// Saturday: работа=подработка 10:00-14:00 (14400s)
		completedSpan("2019-06-15T10:00:00Z", "2019-06-15T14:00:00Z", tag("работа", "подработка")),
	}
}

// completedSpan builds a completed time span owned by user 1.
func completedSpan(from, to string, tags ...model.TimeSpanTag) model.TimeSpan {
	end := test.Time(to)
	return model.TimeSpan{
		StartUserTime: test.Time(from),
		EndUserTime:   &end,
		UserID:        1,
		Tags:          tags,
	}
}

// expectedWeeklyDaily builds the expected Stats2 Daily result for a week whose
// visible series are exactly `present`. Series not in `present` are absent
// entirely (they never survive Include/Exclude), while series that are present
// but have no data on a day are zero-filled. Entries are ordered canonically.
func expectedWeeklyDaily(present []string) []*gqlmodel.RangedStatisticsEntries {
	presentSet := map[string]bool{}
	for _, s := range present {
		presentSet[s] = true
	}
	out := make([]*gqlmodel.RangedStatisticsEntries, 0, len(weekDays))
	for _, day := range weekDays {
		entries := []*gqlmodel.StatisticsEntry{}
		for _, s := range cyrillicSeries {
			if !presentSet[s] {
				continue
			}
			entries = append(entries, &gqlmodel.StatisticsEntry{
				Key:                "работа",
				Value:              s,
				TimeSpendInSeconds: cyrillicDailySecs[day][s],
			})
		}
		out = append(out, result(day+"T00:00:00Z", day+"T23:59:59Z", entries...))
	}
	return out
}

// assertRangesEqual compares the range boundaries (as instants) and the
// per-range entries, normalising entry order (entry order within a range is not
// part of the GraphQL contract, so it must not be asserted).
func assertRangesEqual(t *testing.T, expected, actual []*gqlmodel.RangedStatisticsEntries) {
	require.Len(t, actual, len(expected))
	for i := range expected {
		e, a := expected[i], actual[i]
		require.Truef(t, e.Start.Time().Equal(a.Start.Time()), "range %d start: want %s got %s", i, e.Start, a.Start)
		require.Truef(t, e.End.Time().Equal(a.End.Time()), "range %d end: want %s got %s", i, e.End, a.End)
		require.Lenf(t, a.Entries, len(e.Entries), "range %d entry count", i)
		sortEntries(e.Entries)
		sortEntries(a.Entries)
		require.Equalf(t, e.Entries, a.Entries, "range %d entries", i)
	}
}

func sortEntries(entries []*gqlmodel.StatisticsEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Key != entries[j].Key {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].Value < entries[j].Value
	})
}

// assertWeeklyTotals sums each visible series across all returned ranges and
// compares against the independent hand-computed weekly totals.
func assertWeeklyTotals(t *testing.T, stats []*gqlmodel.RangedStatisticsEntries, present []string) {
	sum := map[string]float64{}
	for _, r := range stats {
		for _, e := range r.Entries {
			sum[e.Value] += e.TimeSpendInSeconds
		}
	}
	for _, s := range present {
		require.InDeltaf(t, cyrillicWeeklyTotals[s], sum[s], 0.5, "weekly total for series %q", s)
	}
}

// seedStats2Week seeds users, the explicit Monday week-start setting and the
// Cyrillic fixture spans, plus an isolation span owned by user 2 (a работа=дом
// span on Monday) that must never leak into user 1's statistics.
func seedStats2Week(t *testing.T, db *gorm.DB) {
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1}}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Name: "u2", Pass: []byte{2}}).Error)
	// Explicit week settings: the week starts on Monday.
	require.NoError(t, db.Create(&model.UserSetting{UserID: 1, FirstDayOfTheWeek: "Monday"}).Error)
	for _, span := range cyrillicSpans() {
		require.NoError(t, db.Create(&span).Error, "seed cyrillic span")
	}
	// Another user's identical-looking span must stay invisible to user 1.
	iso := tsForUser(2, "2019-06-10T09:00:00Z", 3*time.Hour, tag("работа", "дом"))
	require.NoError(t, db.Create(&iso).Error)
}

// stats2Scenario is one Stats2 Daily acceptance scenario over the fixed week.
type stats2Scenario struct {
	name      string
	include   []*gqlmodel.InputTimeSpanTag
	exclude   []*gqlmodel.InputTimeSpanTag
	present   []string
	nonUTCNow string // if set, `now` is passed with this non-UTC offset
}

func stats2Scenarios() []stats2Scenario {
	return []stats2Scenario{
		{name: "full week, three series, zero-fill", present: cyrillicSeries},
		{name: "include дом+офис", present: []string{"дом", "офис"},
			include: []*gqlmodel.InputTimeSpanTag{{Key: "работа", Value: "дом"}, {Key: "работа", Value: "офис"}}},
		{name: "exclude подработка", present: []string{"дом", "офис"},
			exclude: []*gqlmodel.InputTimeSpanTag{{Key: "работа", Value: "подработка"}}},
		{name: "non-UTC now offset (+07:00) preserved via OmitTimeZone", present: cyrillicSeries,
			nonUTCNow: "2019-06-17T00:00:00+07:00"},
	}
}

// runStats2DailySuite runs every Stats2 Daily scenario on the database provided
// by open, optionally forcing a non-UTC PostgreSQL session TimeZone first.
func runStats2DailySuite(t *testing.T, open dbFactory, sessionTZ string) {
	for _, sc := range stats2Scenarios() {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			db, cleanup := open(t)
			defer cleanup()

			if sessionTZ != "" {
				// Fixed constant from the test body, never user input; see the
				// same note in summary_test.go.
				require.NoError(t, db.Exec("SET TIME ZONE '"+sessionTZ+"'").Error)
			}

			seedStats2Week(t, db)

			resolver := ResolverForStatistics{DB: db}

			now := test.ModelTime(weekNow)
			if sc.nonUTCNow != "" {
				now = test.ModelTime(sc.nonUTCNow)
			}

			stats, err := resolver.Stats2(fake.User(1), now, gqlmodel.InputStatsSelection{
				Interval:    gqlmodel.StatsIntervalDaily,
				Tags:        []string{"работа"},
				IncludeTags: sc.include,
				ExcludeTags: sc.exclude,
				Range:       weekRange,
			})
			require.NoError(t, err)
			assertRangesEqual(t, expectedWeeklyDaily(sc.present), stats)
			assertWeeklyTotals(t, stats, sc.present)
		})
	}
}

// TestStats2_daily_sqlite exercises Stats2 Daily on SQLite (always executed).
func TestStats2_daily_sqlite(t *testing.T) {
	runStats2DailySuite(t, func(t *testing.T) (*gorm.DB, func()) {
		db := test.InMemoryDB(t)
		return db.DB, func() { db.Close() }
	}, "")
}

// TestStats2_daily_postgres runs the identical Stats2 Daily suite on a
// disposable PostgreSQL database with the server's default session TimeZone.
func TestStats2_daily_postgres(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	runStats2DailySuite(t, open, "")
}

// TestStats2_daily_postgres_sessionTimeZone runs the identical Stats2 Daily suite
// on PostgreSQL with the session TimeZone forced to a large non-UTC offset
// (Asia/Yekaterinburg, +05). It proves the generated daily range bounds and the
// per-day durations are independent of the PostgreSQL session TimeZone
// (FR-001/FR-002, leveraging WP03's session-TZ fix).
func TestStats2_daily_postgres_sessionTimeZone(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	runStats2DailySuite(t, open, "Asia/Yekaterinburg")
}

// TestMidnightCrossing_manualHalfOpen characterises the difference between
// Stats2's second-inclusive daily builder and manually supplied adjacent
// half-open midnight bounds. With half-open bounds [Fri 00:00, Sat 00:00) and
// [Sat 00:00, Sun 00:00), a span crossing 23:30->00:30 contributes 1800s to
// each side. Stats2's daily builder (see TestStats2_daily_*) instead yields
// 1799s/1800s because Friday's range ends at 23:59:59. Both outcomes are
// preserved and asserted independently so neither is silently changed.
func TestMidnightCrossing_manualHalfOpen(t *testing.T) {
	runMidnightHalfOpenSuite(t, func(t *testing.T) (*gorm.DB, func()) {
		db := test.InMemoryDB(t)
		return db.DB, func() { db.Close() }
	}, "")
}

// TestMidnightCrossing_manualHalfOpen_postgres runs the same half-open midnight
// characterization on a disposable PostgreSQL database.
func TestMidnightCrossing_manualHalfOpen_postgres(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	runMidnightHalfOpenSuite(t, open, "")
}

// TestStats2_single_fractional_sqlite exercises Stats2's single-interval path
// (which delegates straight to Stats) with a static fractional range. The
// response must preserve the original fractional bounds and return the clipped
// per-span rounded seconds, so Stats2 keeps the same lossless identity as Stats
// (FR-003). IntervalSingle returns the raw from/to without the second-inclusive
// daily builder, so fractional bounds flow straight through.
func TestStats2_single_fractional_sqlite(t *testing.T) {
	runStats2SingleFractional(t, func(t *testing.T) (*gorm.DB, func()) {
		db := test.InMemoryDB(t)
		return db.DB, func() { db.Close() }
	}, "")
}

// TestStats2_single_fractional_postgres runs the same Stats2 fractional
// single-interval scenario on a disposable PostgreSQL database.
func TestStats2_single_fractional_postgres(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	runStats2SingleFractional(t, open, "")
}

// TestStats2_single_fractional_postgres_sessionTimeZone runs the same Stats2
// fractional single-interval scenario on PostgreSQL with a non-UTC session
// TimeZone, asserting the fractional response is independent of the session.
func TestStats2_single_fractional_postgres_sessionTimeZone(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	runStats2SingleFractional(t, open, "Asia/Yekaterinburg")
}

// runStats2SingleFractional seeds one completed span and requests a single
// fractional static range through Stats2, asserting the fractional response
// bounds and the independently computed clipped seconds.
func runStats2SingleFractional(t *testing.T, open dbFactory, sessionTZ string) {
	db, cleanup := open(t)
	defer cleanup()
	if sessionTZ != "" {
		require.NoError(t, db.Exec("SET TIME ZONE '"+sessionTZ+"'").Error)
	}
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1}}).Error)
	span := ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))
	require.NoError(t, db.Create(&span).Error)

	resolver := ResolverForStatistics{DB: db}
	stats, err := resolver.Stats2(fake.User(1), test.ModelTime("2019-06-11T10:00:00Z"), gqlmodel.InputStatsSelection{
		Interval: gqlmodel.StatsIntervalSingle,
		Tags:     []string{"type"},
		Range: &gqlmodel.InputRelativeOrStaticRange{
			From: "2019-06-11T10:00:00.200Z",
			To:   "2019-06-11T10:00:01.000Z",
		},
	})
	require.NoError(t, err)
	// Clipped span [10:00:00.200Z, 10:00:01.000Z] = 0.8s, rounded per span = 1s.
	assertRangesEqual(t, []*gqlmodel.RangedStatisticsEntries{
		result("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z",
			&gqlmodel.StatisticsEntry{Key: "type", Value: "review", TimeSpendInSeconds: 1}),
	}, stats)
}

func runMidnightHalfOpenSuite(t *testing.T, open dbFactory, sessionTZ string) {
	db, cleanup := open(t)
	defer cleanup()
	if sessionTZ != "" {
		require.NoError(t, db.Exec("SET TIME ZONE '"+sessionTZ+"'").Error)
	}
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1}}).Error)
	// Completed span crossing Friday midnight.
	midnight := ts("2019-06-14T23:30:00Z", time.Hour, tag("работа", "подработка"))
	require.NoError(t, db.Create(&midnight).Error)

	resolver := ResolverForStatistics{DB: db}
	stats, err := resolver.Stats(fake.User(1),
		[]*gqlmodel.Range{
			rangex("2019-06-14T00:00:00Z", "2019-06-15T00:00:00Z"),
			rangex("2019-06-15T00:00:00Z", "2019-06-16T00:00:00Z"),
		},
		[]string{"работа"}, nil, nil)
	require.NoError(t, err)

	// Half-open adjacent midnight bounds: exactly 1800s on each side.
	assertRangesEqual(t, []*gqlmodel.RangedStatisticsEntries{
		result("2019-06-14T00:00:00Z", "2019-06-15T00:00:00Z",
			&gqlmodel.StatisticsEntry{Key: "работа", Value: "подработка", TimeSpendInSeconds: 1800}),
		result("2019-06-15T00:00:00Z", "2019-06-16T00:00:00Z",
			&gqlmodel.StatisticsEntry{Key: "работа", Value: "подработка", TimeSpendInSeconds: 1800}),
	}, stats)
}
