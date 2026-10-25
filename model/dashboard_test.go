package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDashboardType_Scan(t *testing.T) {
	t.Run("accepts string (PostgreSQL text / SQLite)", func(t *testing.T) {
		var dt DashboardType
		require.NoError(t, dt.Scan("barchart"))
		require.Equal(t, DashboardType("barchart"), dt)
	})

	t.Run("accepts byte slice (PostgreSQL legacy bytea)", func(t *testing.T) {
		var dt DashboardType
		require.NoError(t, dt.Scan([]byte("piechart")))
		require.Equal(t, DashboardType("piechart"), dt)
	})

	t.Run("nil returns defined error", func(t *testing.T) {
		var dt DashboardType
		err := dt.Scan(nil)
		require.ErrorIs(t, err, ErrDashboardTypeNil)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		var dt DashboardType
		err := dt.Scan(42)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported value type")
		require.Contains(t, err.Error(), "expected string or []byte")
	})

	t.Run("value round trips to a string", func(t *testing.T) {
		for _, v := range []DashboardType{TypePieChart, TypeBarChart, TypeLineChart, HorizontalTable, VerticalTable, TypeStackedBarChart} {
			val, err := v.Value()
			require.NoError(t, err)
			require.IsType(t, "", val)
			var back DashboardType
			require.NoError(t, back.Scan(val))
			require.Equal(t, v, back)
		}
	})
}

func TestInterval_Scan(t *testing.T) {
	t.Run("accepts string", func(t *testing.T) {
		var iv Interval
		require.NoError(t, iv.Scan("daily"))
		require.Equal(t, Interval("daily"), iv)
	})

	t.Run("accepts byte slice", func(t *testing.T) {
		var iv Interval
		require.NoError(t, iv.Scan([]byte("weekly")))
		require.Equal(t, Interval("weekly"), iv)
	})

	t.Run("nil returns defined error", func(t *testing.T) {
		var iv Interval
		err := iv.Scan(nil)
		require.ErrorIs(t, err, ErrIntervalNil)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		var iv Interval
		err := iv.Scan(true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported value type")
		require.Contains(t, err.Error(), "expected string or []byte")
	})

	t.Run("value round trips to a string", func(t *testing.T) {
		for _, v := range []Interval{IntervalHourly, IntervalDaily, IntervalWeekly, IntervalMonthly, IntervalYearly, IntervalSingle} {
			val, err := v.Value()
			require.NoError(t, err)
			require.IsType(t, "", val)
			var back Interval
			require.NoError(t, back.Scan(val))
			require.Equal(t, v, back)
		}
	})
}

// Scan must not mutate the receiver when it fails, so a failed scan never leaves
// a partial/wrong enum value behind.
func TestDashboardType_Scan_DoesNotMutateOnError(t *testing.T) {
	var dt DashboardType = TypeBarChart
	require.Error(t, dt.Scan(42))
	require.Equal(t, TypeBarChart, dt)
}
