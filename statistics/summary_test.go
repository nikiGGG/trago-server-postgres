package statistics

import (
	"fmt"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/generated/gqlmodel"
	"github.com/traggo/server/model"
	"github.com/traggo/server/test"
	"github.com/traggo/server/test/fake"
)

func TestSummary_fails(t *testing.T) {
	db := test.InMemoryDB(t)
	defer db.Close()

	resolver := ResolverForStatistics{DB: db.DB}

	_, err := resolver.Stats(fake.User(1), []*gqlmodel.Range{rangex("2019-06-11T10:00:00Z", "2019-06-11T09:00:00Z")}, []string{"type"}, nil, nil)
	require.Error(t, err, "range start must be before range end ")

	// empty range list must be an error, not an empty result
	_, err = resolver.Stats(fake.User(1), []*gqlmodel.Range{}, []string{"type"}, nil, nil)
	require.Error(t, err, "ranges may not be empty")
}

// statsFixtures returns the shared statistics fixtures. The exact same set is
// run against SQLite (TestStats) and, when a disposable PostgreSQL DSN is
// configured, against PostgreSQL 17 (TestStats_postgres). Expected values are
// identical for both engines.
func statsFixtures() []*testEntry {
	return []*testEntry{
		name("simple").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z",
					entry("type", "review", 10*time.Second))),
		name("boundaries 1").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:05Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:05Z",
					entry("type", "review", 5*time.Second))),
		name("boundaries 2").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:05Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:05Z", "2019-06-11T10:00:10Z",
					entry("type", "review", 5*time.Second))),
		name("boundaries 3").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:04Z", "2019-06-11T10:00:06Z")).
			Expected(
				result("2019-06-11T10:00:04Z", "2019-06-11T10:00:06Z",
					entry("type", "review", 2*time.Second))),
		name("exclude 1").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"), tag("issue", "13"))).
			Keys("type").
			Exclude(tag("issue", "13")).
			Ranges(rangex("2019-06-11T10:00:04Z", "2019-06-11T10:00:06Z")).
			Expected(
				result("2019-06-11T10:00:04Z", "2019-06-11T10:00:06Z", []*gqlmodel.StatisticsEntry{}...)),
		name("empty range").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"), tag("issue", "13"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:04Z", "2019-06-11T10:00:06Z"),
				rangex("2019-06-12T10:00:04Z", "2019-06-12T10:00:06Z")).
			Expected(
				result("2019-06-11T10:00:04Z", "2019-06-11T10:00:06Z", entry("type", "review", 2*time.Second)),
				result("2019-06-12T10:00:04Z", "2019-06-12T10:00:06Z", entry("type", "review", 0*time.Second))),
		name("include 1").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"), tag("issue", "13"))).
			Keys("type").
			Include(tag("issue", "13")).
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z",
					entry("type", "review", 10*time.Second))),
		name("multiple 1").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:01:00Z", 10*time.Second, tag("type", "review"), tag("issue", "14")),
			).Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:02:00Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:02:00Z",
					entry("type", "review", 20*time.Second))),
		name("multiple 2").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:01:00Z", 10*time.Second, tag("type", "review"), tag("issue", "14")),
			).Keys("type").
			Exclude(tag("issue", "14")).
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:02:00Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:02:00Z",
					entry("type", "review", 10*time.Second))),
		name("fill empty range").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review")),
			).Keys("type").
			Ranges(
				rangex("2019-06-10T09:58:00Z", "2019-06-10T09:59:59Z"),
				rangex("2019-06-10T10:00:00Z", "2019-06-11T10:02:00Z")).
			Expected(
				result("2019-06-10T09:58:00Z", "2019-06-10T09:59:59Z",
					entry("type", "review", 0*time.Second)),
				result("2019-06-10T10:00:00Z", "2019-06-11T10:02:00Z",
					entry("type", "review", 10*time.Second))),
		name("multiple 3").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:01:00Z", 60*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "12")),
				ts("2019-06-11T10:02:00Z", 7*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:03:00Z", 14*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:04:00Z", 16*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "14")),
				ts("2019-06-11T10:05:00Z", 33*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "10")),
				ts("2019-06-11T10:05:00Z", 12*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "13")),
				ts("2019-06-11T10:06:00Z", 90*time.Second, tag("type", "break")),
				ts("2019-06-11T10:08:00Z", 33*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "10")),
				ts("2019-06-11T10:09:00Z", 21*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:10:00Z", 18*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "12")),
				ts("2019-06-11T10:11:00Z", 56*time.Second, tag("proj", "traggo"), tag("type", "review"), tag("issue", "11")),
				ts("2019-06-11T10:12:00Z", 44*time.Second, tag("type", "break")),
			).Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z",
					entry("type", "break", 134*time.Second),
					entry("type", "review", 133*time.Second),
					entry("type", "support", 45*time.Second),
					entry("type", "work", 102*time.Second))),
		name("multiple 4").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:01:00Z", 60*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "12")),
				ts("2019-06-11T10:02:00Z", 7*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:03:00Z", 14*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:04:00Z", 16*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "14")),
				ts("2019-06-11T10:05:00Z", 33*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "10")),
				ts("2019-06-11T10:05:00Z", 12*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "13")),
				ts("2019-06-11T10:06:00Z", 90*time.Second, tag("type", "break")),
				ts("2019-06-11T10:08:00Z", 32*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "10")),
				ts("2019-06-11T10:09:00Z", 21*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:10:00Z", 18*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "12")),
				ts("2019-06-11T10:11:00Z", 56*time.Second, tag("proj", "traggo"), tag("type", "review"), tag("issue", "11")),
				ts("2019-06-11T10:12:00Z", 44*time.Second, tag("type", "break")),
			).Keys("type", "issue").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z",
					entry("issue", "10", 65*time.Second),
					entry("issue", "11", 56*time.Second),
					entry("issue", "12", 78*time.Second),
					entry("issue", "13", 29*time.Second),
					entry("issue", "14", 16*time.Second),
					entry("issue", "15", 35*time.Second),
					entry("type", "break", 134*time.Second),
					entry("type", "review", 133*time.Second),
					entry("type", "support", 45*time.Second),
					entry("type", "work", 101*time.Second))),
		name("multiple 5").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:01:00Z", 60*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "12")),
				ts("2019-06-11T10:02:00Z", 7*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:03:00Z", 14*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:04:00Z", 16*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "14")),
				ts("2019-06-11T10:05:00Z", 33*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "10")),
				ts("2019-06-11T10:05:00Z", 12*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "13")),
				ts("2019-06-11T10:06:00Z", 90*time.Second, tag("type", "break")),
				ts("2019-06-11T10:08:00Z", 32*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "10")),
				ts("2019-06-11T10:09:00Z", 21*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:10:00Z", 18*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "12")),
				ts("2019-06-11T10:11:00Z", 56*time.Second, tag("proj", "traggo"), tag("type", "review"), tag("issue", "11")),
				ts("2019-06-11T10:12:00Z", 44*time.Second, tag("type", "break")),
			).Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z")).
			Include(tag("issue", "13"), tag("issue", "12")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z",
					entry("type", "review", 77*time.Second),
					entry("type", "support", 12*time.Second),
					entry("type", "work", 18*time.Second))),
		name("multiple 6").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:01:00Z", 60*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "12")),
				ts("2019-06-11T10:02:00Z", 7*time.Second, tag("proj", "gotify"), tag("type", "review"), tag("issue", "13")),
				ts("2019-06-11T10:03:00Z", 14*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:04:00Z", 16*time.Second, tag("proj", "gotify"), tag("type", "work"), tag("issue", "14")),
				ts("2019-06-11T10:05:00Z", 33*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "10")),
				ts("2019-06-11T10:05:00Z", 12*time.Second, tag("proj", "gotify"), tag("type", "support"), tag("issue", "13")),
				ts("2019-06-11T10:06:00Z", 90*time.Second, tag("type", "break")),
				ts("2019-06-11T10:08:00Z", 32*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "10")),
				ts("2019-06-11T10:09:00Z", 21*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "15")),
				ts("2019-06-11T10:10:00Z", 18*time.Second, tag("proj", "traggo"), tag("type", "work"), tag("issue", "12")),
				ts("2019-06-11T10:11:00Z", 56*time.Second, tag("proj", "traggo"), tag("type", "review"), tag("issue", "11")),
				ts("2019-06-11T10:12:00Z", 44*time.Second, tag("type", "break")),
			).Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z")).
			Exclude(tag("issue", "10"), tag("issue", "11"), tag("issue", "14"), tag("issue", "15")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:15:00Z",
					entry("type", "break", 134*time.Second),
					entry("type", "review", 77*time.Second),
					entry("type", "support", 12*time.Second),
					entry("type", "work", 18*time.Second))),
		// ---- T003 boundary characterization (same on SQLite and PostgreSQL) ----
		name("equal range").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:00Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:00Z",
					entry("type", "review", 0*time.Second))),
		name("empty keys").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys().
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z", []*gqlmodel.StatisticsEntry{}...)),
		name("active span excluded").
			Load(active("2019-06-11T10:00:00Z", tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z", []*gqlmodel.StatisticsEntry{}...)),
		name("fractional rounding 0.4s -> 0").
			Load(ts("2019-06-11T10:00:00.600Z", 400*time.Millisecond, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:01Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:01Z",
					entry("type", "review", 0*time.Second))),
		name("fractional rounding 0.8s -> 1").
			Load(ts("2019-06-11T10:00:00.200Z", 800*time.Millisecond, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:01Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:01Z",
					entry("type", "review", 1*time.Second))),
		// ---- WP03/T012 additional regression cases (same on both engines) ----
		// Two requested ranges share the same start but have different ends.
		// Grouping must keep them as separate ranges (group by query_end too),
		// not merge them into a single summed row.
		name("same start different end").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(
				rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:05Z"),
				rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:05Z",
					entry("type", "review", 5*time.Second)),
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z",
					entry("type", "review", 10*time.Second))),
		// Another user's identical span must not leak into user 1's statistics.
		name("user isolation").
			Load(
				ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review")),
				tsForUser(2, "2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review")),
			).Keys("type").
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z",
					entry("type", "review", 10*time.Second))),
		// Exclude wins when the same span matches both Include and Exclude.
		name("exclude precedence").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"), tag("issue", "13"))).
			Keys("type").
			Include(tag("issue", "13")).
			Exclude(tag("issue", "13")).
			Ranges(rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")).
			Expected(
				result("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z", []*gqlmodel.StatisticsEntry{}...)),
		// ---- WP02 fractional-range identity fixtures (FR-003/C-001) ----
		// One completed tagged span 10:00:00Z-10:00:10Z requested at fractional
		// bounds. Every expected second is an INDEPENDENT hand calculation of
		// the clipped, per-span rounded duration (identical on SQLite and
		// PostgreSQL). Several requested ranges fall within the same respective
		// whole seconds (start 10:00:00.x, end 10:00:01.x): before the range_id
		// fix they collapsed into one identity and merged, losing the original
		// bounds and distinct totals. The response must preserve the original
		// fractional bounds and return each range's own total.
		name("fractional same-second identities").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(
				rangex("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z"),
				rangex("2019-06-11T10:00:00.000Z", "2019-06-11T10:00:01.800Z"),
				rangex("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.800Z"),
				rangex("2019-06-11T10:00:00.800Z", "2019-06-11T10:00:01.200Z"),
				rangex("2019-06-11T10:00:00.200000001Z", "2019-06-11T10:00:01.800000001Z")).
			Expected(
				result("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z",
					entry("type", "review", 1*time.Second)),
				result("2019-06-11T10:00:00.000Z", "2019-06-11T10:00:01.800Z",
					entry("type", "review", 2*time.Second)),
				result("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.800Z",
					entry("type", "review", 2*time.Second)),
				result("2019-06-11T10:00:00.800Z", "2019-06-11T10:00:01.200Z",
					entry("type", "review", 0*time.Second)),
				result("2019-06-11T10:00:00.200000001Z", "2019-06-11T10:00:01.800000001Z",
					entry("type", "review", 2*time.Second))),
		// Identical duplicate ranges must each return the original total (no
		// duplicate-join amplification), and a fractional range with no overlap
		// retains the existing zero-fill behaviour.
		name("fractional duplicate and empty").
			Load(ts("2019-06-11T10:00:00Z", 10*time.Second, tag("type", "review"))).
			Keys("type").
			Ranges(
				rangex("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z"),
				rangex("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z"),
				rangex("2019-06-11T10:00:20.200Z", "2019-06-11T10:00:21.800Z")).
			Expected(
				result("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z",
					entry("type", "review", 1*time.Second)),
				result("2019-06-11T10:00:00.200Z", "2019-06-11T10:00:01.000Z",
					entry("type", "review", 1*time.Second)),
				result("2019-06-11T10:00:20.200Z", "2019-06-11T10:00:21.800Z",
					entry("type", "review", 0*time.Second))),
		// Two 0.6s spans in the SAME group distinguish sum-of-rounded (2)
		// from rounded-sum (1). Separate tag values would not constrain this.
		name("fractional sum-of-rounded vs rounded-sum").
			Load(
				ts("2019-06-11T10:00:00.000Z", 600*time.Millisecond, tag("type", "review")),
				ts("2019-06-11T10:00:00.000Z", 600*time.Millisecond, tag("type", "review"))).
			Keys("type").
			Ranges(rangex("2019-06-11T10:00:00.000Z", "2019-06-11T10:00:01.000Z")).
			Expected(
				result("2019-06-11T10:00:00.000Z", "2019-06-11T10:00:01.000Z",
					entry("type", "review", 2*time.Second))),
	}
}

// dbFactory opens a fresh, isolated database (either SQLite in-memory or a
// disposable PostgreSQL database) for one fixture run and returns a cleanup
// function that closes the connection and destroys the database.
type dbFactory func(t *testing.T) (*gorm.DB, func())

// TestStats runs the shared statistics fixtures against SQLite. This is the
// default, always-executed path and must stay green.
func TestStats(t *testing.T) {
	runStatsSuite(t, func(t *testing.T) (*gorm.DB, func()) {
		db := test.InMemoryDB(t)
		return db.DB, func() { db.Close() }
	}, statsFixtures(), "")
}

// TestStats_postgres runs the identical statistics fixtures against a disposable
// PostgreSQL database. It is OPT-IN: it only executes when TRAGGO_TEST_POSTGRES_DSN
// is set, otherwise it skips cleanly. The session TimeZone is left at the server
// default (typically UTC), which is the path the previous WP03 submission was
// verified on.
//
// This is a PASSING regression for the reported PostgreSQL dashboard failure
// (WP03): before the fix the same suite reproduced `pq: syntax error at or near
// "WHERE"`. Now the dialect-portable query runs and returns the same expected
// values as SQLite. In CI, set TRAGGO_TEST_POSTGRES_REQUIRED=1 so an absent DSN
// is a failure.
func TestStats_postgres(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	// NewPostgresDB handles skip-or-fail when no DSN is configured.
	runStatsSuite(t, open, statsFixtures(), "")
}

// TestStats_postgres_sessionTimeZone is the FR-002 / plan-test-matrix regression
// for PostgreSQL session-TimeZone independence (WP03 review Issue 1/2). It runs
// the exact same shared statistics fixtures as the UTC path, but with the
// session TimeZone forced to a large non-UTC offset (Asia/Yekaterinburg, +05).
//
// Before the fix the CTE range bounds were cast to `timestamp without time
// zone`, so PostgreSQL coerced them to timestamptz using the session TimeZone
// and shifted the effective range by +5h, returning empty/incorrect statistics.
// With the bounds now typed `timestamptz` (matching the start_user_time /
// end_user_time columns) the results are identical to UTC: this test asserts the
// same fixed expected seconds and would fail on the pre-fix code.
//
// This is an OPT-IN PostgreSQL test, gated exactly like TestStats_postgres.
func TestStats_postgres_sessionTimeZone(t *testing.T) {
	cfg := test.PostgresConfigFromEnv()
	open := func(t *testing.T) (*gorm.DB, func()) {
		pdb := test.NewPostgresDB(t, cfg)
		return pdb.DB, func() { pdb.Close() }
	}
	// NewPostgresDB handles skip-or-fail when no DSN is configured.
	runStatsSuite(t, open, statsFixtures(), "Asia/Yekaterinburg")
}

// TestStats_errorPropagation verifies FR-004: a real SQL execution failure
// surfaces as a resolver error instead of successful empty statistics. The
// time_spans table is dropped after a successful fixture setup so the statistics
// query fails at execution time, not during seeding.
func TestStats_errorPropagation(t *testing.T) {
	db := test.InMemoryDB(t)
	defer db.Close()

	require.NoError(t, db.Create(&model.User{ID: 1, Name: "u1", Pass: []byte{1, 2, 3}}).Error)

	resolver := ResolverForStatistics{DB: db.DB}

	// Force a genuine execution-time failure by removing the table the query
	// reads from. This happens after fixture setup, so the error is not an
	// artifact of broken seeding.
	require.NoError(t, db.Exec("DROP TABLE time_spans").Error)

	_, err := resolver.Stats(fake.User(1), []*gqlmodel.Range{rangex("2019-06-11T10:00:00Z", "2019-06-11T10:00:10Z")}, []string{"type"}, nil, nil)
	require.Error(t, err, "a failed statistics query must return a resolver error, not empty statistics")
}

// runStatsSuite seeds and asserts every fixture on the database provided by
// open. When sessionTZ is non-empty (PostgreSQL only), it is applied to the
// session before the fixtures run so the same expectations can be asserted
// under a non-UTC session TimeZone. Fixture write errors fail the subtest
// immediately so a failed insert can never masquerade as an empty result.
func runStatsSuite(t *testing.T, open dbFactory, data []*testEntry, sessionTZ string) {
	for _, entry := range data {
		entry := entry
		t.Run(entry.name, func(t *testing.T) {
			db, cleanup := open(t)
			defer cleanup()

			if sessionTZ != "" {
				// SET TIME ZONE accepts a literal name, not a bind parameter
				// (PostgreSQL rejects "SET TIME ZONE $1"). The value is always a
				// fixed constant from the test body, never user input, and timezone
				// names cannot contain a single quote, so inlining in single quotes
				// is safe.
				if err := db.Exec("SET TIME ZONE '" + sessionTZ + "'").Error; err != nil {
					t.Fatalf("set session time zone %q for %q: %v", sessionTZ, entry.name, err)
				}
			}

			// Seed a user for every UserID referenced by the fixture, plus
			// user 1 always (the resolver runs as user 1). User isolation is
			// asserted by fixtures that also load another user's spans.
			userIDs := map[int]struct{}{1: {}}
			for _, timespan := range entry.timespans {
				userIDs[timespan.UserID] = struct{}{}
			}
			for id := range userIDs {
				if err := db.Create(&model.User{ID: id, Name: fmt.Sprintf("test%d", id), Pass: []byte{1, 2, 3}}).Error; err != nil {
					t.Fatalf("seed user %d for %q: %v", id, entry.name, err)
				}
			}
			for _, timespan := range entry.timespans {
				if err := db.Create(&timespan).Error; err != nil {
					t.Fatalf("seed time span for %q: %v", entry.name, err)
				}
			}

			resolver := ResolverForStatistics{DB: db}

			stats, err := resolver.Stats(fake.User(1), entry.ranges, entry.keys, entry.exclude, entry.include)
			require.NoError(t, err)
			require.Equal(t, entry.expected, stats)
			require.Len(t, stats, len(entry.expected))
		})
	}
}

func ts(start string, d time.Duration, tags ...model.TimeSpanTag) model.TimeSpan {
	end := test.Time(start).Add(d)
	return model.TimeSpan{
		StartUserTime: test.Time(start),
		EndUserTime:   &end,
		UserID:        1,
		Tags:          tags,
	}
}

// tsForUser builds a completed time span owned by an arbitrary user, so user
// isolation can be exercised in the shared fixtures.
func tsForUser(userID int, start string, d time.Duration, tags ...model.TimeSpanTag) model.TimeSpan {
	end := test.Time(start).Add(d)
	return model.TimeSpan{
		StartUserTime: test.Time(start),
		EndUserTime:   &end,
		UserID:        userID,
		Tags:          tags,
	}
}

// active builds a running time span (no end) for a user. Running spans are
// expected to be excluded from statistics.
func active(start string, tags ...model.TimeSpanTag) model.TimeSpan {
	return model.TimeSpan{
		StartUserTime: test.Time(start),
		EndUserTime:   nil,
		UserID:        1,
		Tags:          tags,
	}
}

type testEntry struct {
	timespans []model.TimeSpan
	keys      []string
	exclude   []*gqlmodel.InputTimeSpanTag
	include   []*gqlmodel.InputTimeSpanTag
	name      string
	ranges    []*gqlmodel.Range
	expected  []*gqlmodel.RangedStatisticsEntries
}

func name(name string) *testEntry {
	return &testEntry{
		name: name,
	}
}

func (e *testEntry) Expected(expected ...*gqlmodel.RangedStatisticsEntries) *testEntry {
	e.expected = expected
	return e
}

func (e *testEntry) Keys(keys ...string) *testEntry {
	e.keys = keys
	return e
}

func (e *testEntry) Load(timespans ...model.TimeSpan) *testEntry {
	e.timespans = timespans
	return e
}

func (e *testEntry) Ranges(ranges ...*gqlmodel.Range) *testEntry {
	e.ranges = ranges
	return e
}

func (e *testEntry) Include(includes ...model.TimeSpanTag) *testEntry {
	for _, entry := range includes {
		e.include = append(e.include, &gqlmodel.InputTimeSpanTag{
			Key:   entry.Key,
			Value: entry.StringValue,
		})
	}
	return e
}
func (e *testEntry) Exclude(excludes ...model.TimeSpanTag) *testEntry {
	for _, entry := range excludes {
		e.exclude = append(e.exclude, &gqlmodel.InputTimeSpanTag{
			Key:   entry.Key,
			Value: entry.StringValue,
		})
	}
	return e
}

func tag(key string, value string) model.TimeSpanTag {
	return model.TimeSpanTag{
		Key:         key,
		StringValue: value,
	}
}

func rangex(start, stop string) *gqlmodel.Range {
	return &gqlmodel.Range{
		Start: test.ModelTime(start),
		End:   test.ModelTime(stop),
	}
}

func result(start, stop string, entries ...*gqlmodel.StatisticsEntry) *gqlmodel.RangedStatisticsEntries {
	return &gqlmodel.RangedStatisticsEntries{
		Start:   test.ModelTimeUTC(start),
		End:     test.ModelTimeUTC(stop),
		Entries: entries,
	}
}

func entry(key string, value string, duration time.Duration) *gqlmodel.StatisticsEntry {
	return &gqlmodel.StatisticsEntry{
		Key:                key,
		Value:              value,
		TimeSpendInSeconds: duration.Seconds(),
	}
}
