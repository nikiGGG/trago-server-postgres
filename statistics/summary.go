package statistics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/traggo/server/auth"
	"github.com/traggo/server/generated/gqlmodel"
	"github.com/traggo/server/model"
)

// Stats groups the time spans by tag key.
//
// The query is assembled from dialect-specific fragments so the same logical
// statistics query runs on SQLite and PostgreSQL. PostgreSQL needs explicit
// timestamp typing for the range bounds (VALUES parameters otherwise resolve
// as text) and a different duration expression (ROUND(EXTRACT(EPOCH FROM ...))
// instead of julianday arithmetic). Every requested range carries an integer
// range_id (its input position) that becomes the aggregate identity, so range
// identity is decoupled from how the database stores or formats the bounds;
// response bounds are rebuilt from the original Go input. Every user-supplied
// value stays bound as a parameter; no user input is concatenated into the SQL.
func (r *ResolverForStatistics) Stats(ctx context.Context, ranges []*gqlmodel.Range, tags []string, excludeTags []*gqlmodel.InputTimeSpanTag, requireTags []*gqlmodel.InputTimeSpanTag) ([]*gqlmodel.RangedStatisticsEntries, error) {
	if len(ranges) == 0 {
		return nil, errors.New("ranges may not be empty")
	}
	var variables []interface{}
	var rangesStrs []string
	for i, rg := range ranges {
		if rg.Start.Time().After(rg.End.Time()) {
			return nil, fmt.Errorf("range start must be before range end")
		}
		rangesStrs = append(rangesStrs, r.boundRow(i))
		variables = append(variables, rg.Start.OmitTimeZone(), rg.End.OmitTimeZone())
	}
	queryRanges := strings.Join(rangesStrs, ", ")

	queryRequire, requireVars := build(requireTags, "1 = 1")
	variables = append(variables, requireVars...)

	queryExclude, excludeVars := build(excludeTags, "1 != 1")
	variables = append(variables, excludeVars...)

	query := fmt.Sprintf(`
WITH dates(query_start, query_end, range_id) AS (
    VALUES %s
)
SELECT dates.range_id                              as range_id,
       tst.key                                     as key,
       tst.string_value                            as string_value,
       %s                                          as time_spend_in_seconds
FROM dates
         CROSS JOIN time_span_tags as tst
         JOIN (
    SELECT tsx.id, tsx.start_user_time, tsx.end_user_time, tsx.user_id
    FROM time_spans as tsx
    WHERE (EXISTS(SELECT 1
                  FROM time_span_tags as tstx
                  WHERE (tstx.time_span_id = tsx.id)
                    AND (%s)))
      AND (NOT EXISTS(SELECT 1
                      FROM time_span_tags as tstx
                      WHERE (tstx.time_span_id = tsx.id)
                        AND (%s)))
) as ts on ts.id = tst.time_span_id
WHERE (tst.key in (?))
  AND (ts.user_id = ?)
  AND (ts.start_user_time <= dates.query_end AND ts.end_user_time >= dates.query_start)
GROUP BY dates.range_id,
         tst.key,
         tst.string_value;
`, queryRanges, r.durationExpr(), queryRequire, queryExclude)
	variables = append(variables, tags, auth.GetUser(ctx).ID)

	var entries []statReturn
	queryHandle := r.DB.Raw(query, variables...).Scan(&entries)
	if err := queryHandle.Error; err != nil {
		return nil, err
	}

	statisticsEntries := buildEmptyRanges(ranges)
	attachAggregates(statisticsEntries, entries)
	fillEmptyTags(statisticsEntries)

	return statisticsEntries, nil
}

// isPostgres reports whether the active database is PostgreSQL. All other
// dialects (notably SQLite) use the original julianday-based fragments.
func (r *ResolverForStatistics) isPostgres() bool {
	return r.DB.Dialect().GetName() == "postgres"
}

// boundRow returns one CTE VALUES row for a single query range at the given
// input position idx. The third column is a per-input-position integer range_id
// that becomes the aggregate identity: it is independent of the database
// timestamp representation, so distinct requested ranges (including fractional
// bounds finer than PostgreSQL's timestamp precision) can never merge, and
// duplicate ranges each keep their own total. The range_id is a trusted loop
// index, never user input, so it is emitted as an integer literal. PostgreSQL
// explicitly types the two bound placeholders as "timestamp with time zone"
// (timestamptz) so the values do not resolve as text, and so the comparison
// against the local user-time timestamptz columns performs no session-TimeZone
// coercion. The bound parameter is rg.Start.OmitTimeZone() (a UTC-located
// absolute instant), which lib/pq encodes with its offset; casting that to
// timestamptz preserves the absolute instant under any session TimeZone.
// SQLite keeps the original untyped row.
func (r *ResolverForStatistics) boundRow(idx int) string {
	if r.isPostgres() {
		return fmt.Sprintf("(?::timestamptz, ?::timestamptz, %d)", idx)
	}
	return fmt.Sprintf("(?, ?, %d)", idx)
}

// durationExpr returns the per-span duration expression used inside sum(). It
// clips each span to the requested range, then rounds the whole-second duration
// per span before summation. PostgreSQL computes the clipped interval in seconds
// with ROUND(EXTRACT(EPOCH FROM ...)); SQLite keeps the original rounded
// julianday arithmetic. Both round each span to whole seconds before summing.
func (r *ResolverForStatistics) durationExpr() string {
	if r.isPostgres() {
		return `sum(round(extract(epoch from ` +
			`((CASE WHEN ts.end_user_time > dates.query_end THEN dates.query_end ELSE ts.end_user_time END) ` +
			`- (CASE WHEN ts.start_user_time < dates.query_start THEN dates.query_start ELSE ts.start_user_time END)))))`
	}
	return `sum(round((julianday(CASE WHEN ts.end_user_time > dates.query_end THEN dates.query_end ELSE ts.end_user_time END) - ` +
		`julianday(CASE WHEN ts.start_user_time < dates.query_start THEN dates.query_start ELSE ts.start_user_time END)) ` +
		`* 24 * 60 * 60, 0))`
}

func fillEmptyTags(statisticsEntries []*gqlmodel.RangedStatisticsEntries) {
	lookup := make(map[string]struct{})
	for _, entry := range statisticsEntries {
		for _, statEntry := range entry.Entries {
			lookup[key(statEntry.Key, statEntry.Value)] = struct{}{}
		}
	}
	for _, entry := range statisticsEntries {
		existing := make(map[string]struct{}, len(lookup))
		for _, statEntry := range entry.Entries {
			existing[key(statEntry.Key, statEntry.Value)] = struct{}{}
		}
		for identifier := range lookup {
			if _, ok := existing[identifier]; !ok {
				key, value := extract(identifier)
				entry.Entries = append(entry.Entries, &gqlmodel.StatisticsEntry{Key: key, Value: value, TimeSpendInSeconds: 0})
			}
		}
	}
}

// buildEmptyRanges builds one response range per requested range, using the
// original Go bounds (UTC convention) and an empty entry list. Response bounds
// therefore never depend on how the database stored or formatted the bounds;
// accepted fractional precision is preserved even when PostgreSQL stores less
// precision than the input.
func buildEmptyRanges(ranges []*gqlmodel.Range) []*gqlmodel.RangedStatisticsEntries {
	result := make([]*gqlmodel.RangedStatisticsEntries, len(ranges))
	for i, rg := range ranges {
		result[i] = &gqlmodel.RangedStatisticsEntries{
			Start:   model.Time(rg.Start.UTC()),
			End:     model.Time(rg.End.UTC()),
			Entries: []*gqlmodel.StatisticsEntry{},
		}
	}
	return result
}

// attachAggregates appends each scanned aggregate row to the response range
// identified by its per-input-position range_id. The response ranges were built
// from the original Go bounds, so a row is attached to exactly the requested
// range it belongs to; distinct fractional ranges (including those finer than
// the database timestamp precision) stay distinct, and duplicate ranges each
// receive their own total without double-counting.
func attachAggregates(result []*gqlmodel.RangedStatisticsEntries, entries []statReturn) {
	for _, entry := range entries {
		if entry.RangeID < 0 || entry.RangeID >= len(result) {
			continue
		}
		result[entry.RangeID].Entries = append(result[entry.RangeID].Entries, &gqlmodel.StatisticsEntry{
			Key:                entry.Key,
			Value:              entry.StringValue,
			TimeSpendInSeconds: entry.TimeSpendInSeconds,
		})
	}
}

type statReturn struct {
	RangeID            int
	Key                string
	StringValue        string
	TimeSpendInSeconds float64
}

func build(tags []*gqlmodel.InputTimeSpanTag, noop string) (string, []interface{}) {
	if len(tags) == 0 {
		return noop, nil
	}

	var have []string
	var haveParams []interface{}
	for _, tag := range tags {
		have = append(have, "(tstx.key = ? AND tstx.string_value = ?)")
		haveParams = append(haveParams, tag.Key, tag.Value)
	}

	return strings.Join(have, " OR "), haveParams
}

func key(key string, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

func extract(id string) (string, string) {
	split := strings.Split(id, ":")
	return split[0], split[1]
}
