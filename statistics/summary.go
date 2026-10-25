package statistics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/traggo/server/auth"
	"github.com/traggo/server/generated/gqlmodel"
	"github.com/traggo/server/model"
)

// Stats groups the time spans by tag key.
//
// The query is assembled from dialect-specific fragments so the same logical
// statistics query runs on SQLite and PostgreSQL. PostgreSQL needs explicit
// timestamp typing for the range bounds (VALUES parameters otherwise resolve
// as text), a different duration expression (ROUND(EXTRACT(EPOCH FROM ...))
// instead of julianday arithmetic) and a text-normalized range bound so the
// shared result grouping can parse it. SQLite keeps its original julianday
// behavior. Every user-supplied value stays bound as a parameter; no user input
// is concatenated into the SQL.
func (r *ResolverForStatistics) Stats(ctx context.Context, ranges []*gqlmodel.Range, tags []string, excludeTags []*gqlmodel.InputTimeSpanTag, requireTags []*gqlmodel.InputTimeSpanTag) ([]*gqlmodel.RangedStatisticsEntries, error) {
	if len(ranges) == 0 {
		return nil, errors.New("ranges may not be empty")
	}
	var variables []interface{}
	var rangesStrs []string
	for _, rg := range ranges {
		if rg.Start.Time().After(rg.End.Time()) {
			return nil, fmt.Errorf("range start must be before range end")
		}
		rangesStrs = append(rangesStrs, r.boundRow())
		variables = append(variables, rg.Start.OmitTimeZone(), rg.End.OmitTimeZone())
	}
	queryRanges := strings.Join(rangesStrs, ", ")

	queryRequire, requireVars := build(requireTags, "1 = 1")
	variables = append(variables, requireVars...)

	queryExclude, excludeVars := build(excludeTags, "1 != 1")
	variables = append(variables, excludeVars...)

	query := fmt.Sprintf(`
WITH dates(query_start, query_end) AS (
    VALUES %s
)
SELECT %s                                          as query_start,
       %s                                          as query_end,
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
GROUP BY dates.query_start,
         dates.query_end,
         tst.key,
         tst.string_value;
`, queryRanges, r.boundStartExpr(), r.boundEndExpr(), r.durationExpr(), queryRequire, queryExclude)
	variables = append(variables, tags, auth.GetUser(ctx).ID)

	var entries []statReturn
	queryHandle := r.DB.Raw(query, variables...).Scan(&entries)
	if err := queryHandle.Error; err != nil {
		return nil, err
	}

	statisticsEntries, err := group(entries)
	if err != nil {
		return nil, err
	}

	statisticsEntries = fillEmptyRanges(ranges, statisticsEntries)
	fillEmptyTags(statisticsEntries)

	return statisticsEntries, nil
}

// isPostgres reports whether the active database is PostgreSQL. All other
// dialects (notably SQLite) use the original julianday-based fragments.
func (r *ResolverForStatistics) isPostgres() bool {
	return r.DB.Dialect().GetName() == "postgres"
}

// boundRow returns one CTE VALUES row for a single query range. PostgreSQL
// explicitly types the two bound placeholders as "timestamp with time zone"
// (timestamptz) so the values do not resolve as text, and so the comparison
// against the local user-time timestamptz columns performs no session-TimeZone
// coercion. The bound parameter is rg.Start.OmitTimeZone() (a UTC-located
// absolute instant), which lib/pq encodes with its offset; casting that to
// timestamptz preserves the absolute instant under any session TimeZone.
// SQLite keeps the original untyped row.
func (r *ResolverForStatistics) boundRow() string {
	if r.isPostgres() {
		return "(?::timestamptz, ?::timestamptz)"
	}
	return "(?, ?)"
}

// boundStartExpr returns the SQL expression producing the range start as the
// text form the shared result grouping parses. PostgreSQL timestamps scan back
// as RFC3339 (with a 'T' separator), which the grouping layout cannot parse, so
// the bound is normalized to the same 'YYYY-MM-DD HH24:MI:SS+00:00' text that
// SQLite returns for the raw bound parameter. The bound is now a timestamptz,
// so AT TIME ZONE 'UTC' is required to render its wall-clock in UTC; the
// user-time values carry no real timezone and are interpreted as UTC, so
// '+00:00' is appended literally.
func (r *ResolverForStatistics) boundStartExpr() string {
	if r.isPostgres() {
		return `(to_char(dates.query_start AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') || '+00:00')`
	}
	return "dates.query_start"
}

// boundEndExpr is the equivalent of boundStartExpr for the range end.
func (r *ResolverForStatistics) boundEndExpr() string {
	if r.isPostgres() {
		return `(to_char(dates.query_end AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') || '+00:00')`
	}
	return "dates.query_end"
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

func fillEmptyRanges(ranges []*gqlmodel.Range, statisticsEntries []*gqlmodel.RangedStatisticsEntries) []*gqlmodel.RangedStatisticsEntries {
	result := make([]*gqlmodel.RangedStatisticsEntries, len(ranges))
	statsIdx := 0
	for idx, r := range ranges {
		if statsIdx < len(statisticsEntries) &&
			statisticsEntries[statsIdx].Start.UTC().Equal(r.Start.UTC()) &&
			statisticsEntries[statsIdx].End.UTC().Equal(r.End.UTC()) {
			result[idx] = statisticsEntries[statsIdx]
			statsIdx++
			continue
		}

		result[idx] = &gqlmodel.RangedStatisticsEntries{
			Start:   model.Time(r.Start.UTC()),
			End:     model.Time(r.End.UTC()),
			Entries: []*gqlmodel.StatisticsEntry{},
		}
	}
	return result
}

type statReturn struct {
	QueryStart         string
	QueryEnd           string
	Key                string
	StringValue        string
	TimeSpendInSeconds float64
}

func group(entries []statReturn) ([]*gqlmodel.RangedStatisticsEntries, error) {
	stats := map[string]*gqlmodel.RangedStatisticsEntries{}
	for _, entry := range entries {
		id := entry.QueryStart + "/" + entry.QueryEnd
		if _, ok := stats[id]; !ok {
			start, err := time.ParseInLocation("2006-01-02 15:04:05Z07:00", entry.QueryStart, time.UTC)
			if err != nil {
				return nil, err
			}
			end, err := time.ParseInLocation("2006-01-02 15:04:05Z07:00", entry.QueryEnd, time.UTC)
			if err != nil {
				return nil, err
			}
			stats[id] = &gqlmodel.RangedStatisticsEntries{Start: model.Time(start), End: model.Time(end)}
		}
		stats[id].Entries = append(stats[id].Entries, &gqlmodel.StatisticsEntry{
			Key:                entry.Key,
			Value:              entry.StringValue,
			TimeSpendInSeconds: entry.TimeSpendInSeconds,
		})
	}

	var result []*gqlmodel.RangedStatisticsEntries
	for _, value := range stats {
		result = append(result, value)
	}

	// Order by range start, then range end. A plain "Start before End"
	// comparison is not a strict ordering and misbehaves when two requested
	// ranges share the same start (different ends), which the shared grouping
	// and zero-fill rely on.
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].Start.Time().Equal(result[j].Start.Time()) {
			return result[i].Start.Time().Before(result[j].Start.Time())
		}
		return result[i].End.Time().Before(result[j].End.Time())
	})
	return result, nil
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
