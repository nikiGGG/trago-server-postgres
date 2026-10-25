package model

// All returns all schema instances, ordered so every foreign-key reference
// points at a table that already exists when GORM AutoMigrate creates it.
//
// PostgreSQL rejects forward REFERENCES at CREATE TABLE time (SQLite tolerates
// them). The order below is a topological order of the FK dependency graph so
// that production startup migration (database.New -> AutoMigrate(model.All()...))
// works on a fresh PostgreSQL database without any manual table-first workaround:
//
//	users -> {tag_definitions, devices, time_spans, user_settings, dashboards}
//	time_spans -> time_span_tags
//	dashboards -> {dashboard_entries, dashboard_ranges}
//	dashboard_entries -> dashboard_tag_filters
func All() []interface{} {
	return []interface{}{
		new(User),
		new(TagDefinition),
		new(Device),
		new(TimeSpan),
		new(TimeSpanTag),
		new(UserSetting),
		new(Dashboard),
		new(DashboardEntry),
		new(DashboardTagFilter),
		new(DashboardRange),
	}
}
