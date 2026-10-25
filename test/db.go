package test

import (
	"fmt"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/traggo/server/database"
	"github.com/traggo/server/model"
)

// PostgreSQL-backed tests (see postgres.go) are opt-in. They only run when a
// disposable test DSN is configured through PostgresConfigFromEnv. The DSN must
// point at a test-only PostgreSQL server; the harness never connects to the
// database named in the DSN for fixtures - it only uses that maintenance
// connection to CREATE/DROP a brand-new random database per test.

// Environment variables controlling the PostgreSQL test harness.
const (
	// PostgresDSNEnv holds a lib/pq connection string to a disposable, test-only
	// PostgreSQL server. When non-empty, PostgreSQL-backed suites execute
	// instead of skipping. Example:
	//   postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable
	// The database named in the DSN (commonly "postgres") is used only as a
	// maintenance target to run CREATE DATABASE / DROP DATABASE. Never point
	// this at a production or shared database.
	PostgresDSNEnv = "TRAGGO_TEST_POSTGRES_DSN"
	// PostgresRequiredEnv, when set to a truthy value (1/true/yes/on), turns a
	// missing TRAGGO_TEST_POSTGRES_DSN (or zero executed PostgreSQL tests) into
	// a hard failure instead of a skip. Intended for CI so that the PostgreSQL
	// job cannot silently pass without running real fixtures.
	PostgresRequiredEnv = "TRAGGO_TEST_POSTGRES_REQUIRED"
)

// PostgresConfig holds the configuration for a disposable PostgreSQL test run.
type PostgresConfig struct {
	// BaseDSN is the disposable server DSN from PostgresDSNEnv (empty if unset).
	BaseDSN string
	// Required mirrors PostgresRequiredEnv: absence becomes failure not skip.
	Required bool
}

// PostgresConfigFromEnv reads the PostgreSQL test configuration from the
// environment. It never returns a production default - an unset DSN means the
// PostgreSQL suite is simply unavailable (Enabled() == false).
func PostgresConfigFromEnv() PostgresConfig {
	return PostgresConfig{
		BaseDSN:  strings.TrimSpace(os.Getenv(PostgresDSNEnv)),
		Required: envTruthy(PostgresRequiredEnv),
	}
}

// Enabled reports whether a disposable PostgreSQL test DSN is configured.
func (c PostgresConfig) Enabled() bool {
	return c.BaseDSN != ""
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// InMemoryDB create a in memory database for testing.
func InMemoryDB(t assert.TestingT) *Database {
	db, err := database.New("sqlite3", "file::memory:?mode=memory&cache=shared")
	assert.Nil(t, err)
	return &Database{DB: db, t: t}
}

// Database wraps the gorm.DB and provides helper methods
type Database struct {
	*gorm.DB
	t assert.TestingT
}

// User creates a user
func (d *Database) User(id int) *UserWithDatabase {
	user := d.NewUser(id, fmt.Sprint("test", id), false)
	return &UserWithDatabase{
		User: user,
		DB:   d.DB,
		t:    d.t,
	}
}

// UserWithDatabase wraps gorm.DB and provides helper methods
type UserWithDatabase struct {
	User model.User
	*gorm.DB
	t assert.TestingT
}

// NewDevice creates a device.
func (d *UserWithDatabase) NewDevice(id int, token string, name string) model.Device {
	device := model.Device{
		ID:        id,
		Token:     token,
		UserID:    d.User.ID,
		Name:      name,
		Type:      model.TypeNoExpiry,
		ActiveAt:  Time("2009-06-30T18:30:00Z"),
		CreatedAt: Time("2009-06-30T18:30:00Z"),
	}
	d.Create(&device)
	return device
}

// NewTagDefinition creates a tag definition.
func (d *UserWithDatabase) NewTagDefinition(key string) model.TagDefinition {
	tagDefinition := model.TagDefinition{
		UserID: d.User.ID,
		Key:    key,
	}
	d.Create(&tagDefinition)
	return tagDefinition
}

// AssertHasTagDefinition asserts if a tag definition exists.
func (d *UserWithDatabase) AssertHasTagDefinition(key string, exist bool) *UserWithDatabase {
	existActual := !d.DB.
		Where(&model.TagDefinition{Key: key, UserID: d.User.ID}).
		Find(new(model.TagDefinition)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// AssertHasDevice asserts if a device exists.
func (d *UserWithDatabase) AssertHasDevice(name string, exist bool) *UserWithDatabase {
	existActual := !d.DB.
		Where(&model.Device{Name: name, UserID: d.User.ID}).
		Find(new(model.Device)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// NewUser creates a user
func (d *Database) NewUser(id int, name string, admin bool) model.User {
	return d.NewUserPass(id, name, []byte{1, 2, 3}, admin)
}

// NewUserPass creates a user
func (d *Database) NewUserPass(id int, name string, pass []byte, admin bool) model.User {
	user := model.User{
		ID:    id,
		Name:  name,
		Pass:  pass,
		Admin: admin,
	}
	d.Create(&user)
	return user
}

// AssertExists asserts if the tag exists or not.
func (d *UserWithDatabase) AssertExists(exist bool) *UserWithDatabase {
	existActual := !d.DB.
		Where(&model.User{ID: d.User.ID}).
		Find(new(model.User)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// RunningTimeSpan adds a time span without end.
func (d *UserWithDatabase) RunningTimeSpan(from string) *TimeSpanWithDatabase {
	timeSpan := model.TimeSpan{
		UserID:        d.User.ID,
		StartUserTime: ModelTime(from).OmitTimeZone(),
		StartUTC:      ModelTime(from).UTC(),
	}

	d.DB.Create(&timeSpan)

	return &TimeSpanWithDatabase{
		User:     d.User,
		TimeSpan: timeSpan,
		DB:       d.DB,
		t:        d.t,
	}
}

// Dashboard adds a dashboard.
func (d *UserWithDatabase) Dashboard(name string) *DashboardWithDatabase {
	dashboard := model.Dashboard{
		UserID: d.User.ID,
		Name:   name,
	}

	d.DB.Create(&dashboard)

	return &DashboardWithDatabase{
		User:      d.User,
		Dashboard: dashboard,
		DB:        d.DB,
		t:         d.t,
	}
}

// Range adds a range to the dashboard.
func (d *DashboardWithDatabase) Range(name string) *DashboardWithDatabase {
	dbRange := model.DashboardRange{
		DashboardID: d.Dashboard.ID,
		Name:        name,
		From:        "now-1d",
		To:          "now",
	}
	d.Dashboard.Ranges = append(d.Dashboard.Ranges, dbRange)
	d.DB.Save(&dbRange)
	return d
}

// Entry adds an entry to the dashboard.
func (d *DashboardWithDatabase) Entry(name string) *DashboardWithDatabase {
	entry := model.DashboardEntry{
		DashboardID: d.Dashboard.ID,
		Title:       name,
	}
	d.Dashboard.Entries = append(d.Dashboard.Entries, entry)
	d.DB.Save(&entry)
	return d
}

// AssertExists asserts if the dashboard exists or not.
func (d *DashboardWithDatabase) AssertExists(exist bool) *DashboardWithDatabase {
	existActual := !d.DB.
		Where(&model.Dashboard{ID: d.Dashboard.ID}).
		Find(new(model.Dashboard)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// AssertHasEntry asserts if the entry exists or not.
func (d *DashboardWithDatabase) AssertHasEntry(name string, exist bool) *DashboardWithDatabase {
	existActual := !d.DB.
		Where(&model.DashboardEntry{DashboardID: d.Dashboard.ID, Title: name}).
		Find(new(model.DashboardEntry)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// AssertHasRange asserts if the range exists or not.
func (d *DashboardWithDatabase) AssertHasRange(name string, exist bool) *DashboardWithDatabase {
	existActual := !d.DB.
		Where(&model.DashboardRange{DashboardID: d.Dashboard.ID, Name: name}).
		Find(new(model.DashboardRange)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// TimeSpan adds a time span.
func (d *UserWithDatabase) TimeSpan(from string, to string) *TimeSpanWithDatabase {
	wrapper := d.RunningTimeSpan(from)
	userTime := ModelTime(to).OmitTimeZone()
	wrapper.TimeSpan.EndUserTime = &userTime
	utc := ModelTime(to).UTC()
	wrapper.TimeSpan.EndUTC = &utc
	d.DB.Save(&wrapper.TimeSpan)
	return wrapper
}

// TimeSpanWithDatabase wraps gorm.DB and provides helper methods
type TimeSpanWithDatabase struct {
	User     model.User
	TimeSpan model.TimeSpan
	t        assert.TestingT
	*gorm.DB
}

// AssertHasTag asserts if the tag exists or not.
func (d *TimeSpanWithDatabase) AssertHasTag(key, value string, exist bool) *TimeSpanWithDatabase {
	existActual := !d.DB.
		Where(&model.TimeSpanTag{Key: key, StringValue: value, TimeSpanID: d.TimeSpan.ID}).
		Find(new(model.TimeSpanTag)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// AssertHasTagIgnoreValue asserts if the tag exists or not.
func (d *TimeSpanWithDatabase) AssertHasTagIgnoreValue(key string, exist bool) *TimeSpanWithDatabase {
	existActual := !d.DB.
		Where(&model.TimeSpanTag{Key: key, TimeSpanID: d.TimeSpan.ID}).
		Find(new(model.TimeSpanTag)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// AssertExists asserts if the tag exists or not.
func (d *TimeSpanWithDatabase) AssertExists(exist bool) *TimeSpanWithDatabase {
	existActual := !d.DB.
		Where(&model.TimeSpan{ID: d.TimeSpan.ID}).
		Find(new(model.TimeSpan)).
		RecordNotFound()
	assert.True(d.t, exist == existActual)
	return d
}

// Tag adds a tag to the time span.
func (d *TimeSpanWithDatabase) Tag(key string, stringValue string) *TimeSpanWithDatabase {
	tag := model.TimeSpanTag{
		TimeSpanID:  d.TimeSpan.ID,
		Key:         key,
		StringValue: stringValue,
	}
	d.TimeSpan.Tags = append(d.TimeSpan.Tags, tag)
	d.DB.Save(tag)
	return d
}

// DashboardWithDatabase wraps gorm.DB and provides helper methods
type DashboardWithDatabase struct {
	User      model.User
	Dashboard model.Dashboard
	t         assert.TestingT
	*gorm.DB
}
