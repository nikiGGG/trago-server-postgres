package model

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

// scanStringEnum scans a database value into a string-backed enum field. It
// accepts the two representations used by the supported engines: a Go string
// (PostgreSQL text columns, SQLite) and a byte slice (PostgreSQL legacy bytea
// columns, lib/pq). nil and any other driver type produce a descriptive error
// so that a NULL/unsupported cell can never silently become a wrong value.
func scanStringEnum(name string, out *string, value interface{}) error {
	switch v := value.(type) {
	case nil:
		return fmt.Errorf("%s: cannot scan NULL (nil value)", name)
	case string:
		*out = v
		return nil
	case []byte:
		*out = string(v)
		return nil
	default:
		return fmt.Errorf("%s: unsupported value type %T (%#v), expected string or []byte", name, value, value)
	}
}

// Enum scan errors surfaced for nil input, matching the documented FR-005
// contract ("establish and test nil handling").
var (
	// ErrDashboardTypeNil is returned when a NULL dashboard type is scanned.
	ErrDashboardTypeNil = errors.New("dashboard type: cannot scan NULL (nil value)")
	// ErrIntervalNil is returned when a NULL interval is scanned.
	ErrIntervalNil = errors.New("interval: cannot scan NULL (nil value)")
)

// DashboardRange a named range of a dashboard.
type DashboardRange struct {
	ID          int `gorm:"primary_key;unique_index;AUTO_INCREMENT"`
	Name        string
	DashboardID int `gorm:"type:int REFERENCES dashboards(id) ON DELETE CASCADE"`
	Editable    bool
	From        string
	To          string
}

// Dashboard a dashboard
type Dashboard struct {
	ID      int `gorm:"primary_key;unique_index;AUTO_INCREMENT"`
	UserID  int `gorm:"type:int REFERENCES users(id) ON DELETE CASCADE"`
	Name    string
	Entries []DashboardEntry
	Ranges  []DashboardRange
}

// DashboardEntry an entry which represents a diagram in a dashboard.
type DashboardEntry struct {
	ID          int `gorm:"primary_key;unique_index;AUTO_INCREMENT"`
	DashboardID int `gorm:"type:int REFERENCES dashboards(id) ON DELETE CASCADE"`
	Title       string
	Total       bool `gorm:"default:false"`
	Type        DashboardType
	Keys        string
	Interval    Interval
	RangeID     int
	RangeFrom   string
	RangeTo     string
	TagFilters  []DashboardTagFilter

	MobilePosition  string
	DesktopPosition string
}

// DashboardTagFilter a tag for filtering timespans
type DashboardTagFilter struct {
	DashboardEntryID int `gorm:"type:int REFERENCES dashboard_entries(id) ON DELETE CASCADE"`
	Key              string
	StringValue      string
	Include          bool
}

// DashboardType the dashboard type
type DashboardType string

// Value for db
func (t DashboardType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan for db
func (t *DashboardType) Scan(value interface{}) error {
	if value == nil {
		return ErrDashboardTypeNil
	}
	s := (*string)(t)
	return scanStringEnum("dashboard type", s, value)
}

// Interval the interval in which the diagram data should be grouped.
type Interval string

// Value for db
func (t Interval) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan for db
func (t *Interval) Scan(value interface{}) error {
	if value == nil {
		return ErrIntervalNil
	}
	s := (*string)(t)
	return scanStringEnum("interval", s, value)
}

// No lint errors please.
const (
	TypePieChart        DashboardType = "piechart"
	TypeBarChart        DashboardType = "barchart"
	TypeLineChart       DashboardType = "linechart"
	HorizontalTable     DashboardType = "horizontaltable"
	VerticalTable       DashboardType = "verticaltable"
	TypeStackedBarChart DashboardType = "stackedbarchart"

	IntervalHourly  Interval = "hourly"
	IntervalDaily   Interval = "daily"
	IntervalWeekly  Interval = "weekly"
	IntervalMonthly Interval = "monthly"
	IntervalYearly  Interval = "yearly"
	IntervalSingle  Interval = "single"

	NoRangeIDDefined = -1
)
