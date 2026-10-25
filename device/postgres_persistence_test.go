package device

import (
	"fmt"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/traggo/server/database"
	"github.com/traggo/server/model"
	"github.com/traggo/server/test"
)

func devicePostgresConfig(t *testing.T) test.PostgresConfig {
	cfg := test.PostgresConfigFromEnv()
	if !cfg.Enabled() {
		if cfg.Required {
			t.Fatalf("%s not set but %s=1; PostgreSQL tests are required", test.PostgresDSNEnv, test.PostgresRequiredEnv)
		}
		t.Skipf("%s not set; skipping PostgreSQL device persistence test", test.PostgresDSNEnv)
	}
	return cfg
}

func deviceDSNForDatabase(baseDSN, dbName string) string {
	u, err := url.Parse(baseDSN)
	if err != nil {
		return baseDSN
	}
	u.Path = "/" + dbName
	return u.String()
}

func seedDeviceUser(t *testing.T, db *gorm.DB) {
	require.NoError(t, db.Create(&model.User{ID: 1, Name: "dev-u", Pass: []byte{1}}).Error)
}

func newDevice(id int) model.Device {
	return model.Device{
		ID:        id,
		Token:     fmt.Sprintf("tok-%d", id),
		Name:      fmt.Sprintf("dev-%d", id),
		UserID:    1,
		Type:      model.TypeNoExpiry,
		ActiveAt:  time.Date(2009, 6, 30, 18, 30, 0, 0, time.UTC),
		CreatedAt: time.Date(2009, 6, 30, 18, 30, 0, 0, time.UTC),
	}
}

// ============================================================
// T006: device type persistence on PostgreSQL text AND bytea
// ============================================================

func TestDevicePersistence_text(t *testing.T) {
	cfg := devicePostgresConfig(t)
	pdb := test.NewPostgresDB(t, cfg)
	defer pdb.Close()

	seedDeviceUser(t, pdb.DB)

	dev := newDevice(1)
	require.NoError(t, pdb.Create(&dev).Error, "insert device on text schema")

	var got model.Device
	require.NoError(t, pdb.First(&got, 1).Error)
	require.Equal(t, model.TypeNoExpiry, got.Type)

	require.NoError(t, pdb.Model(&got).Update("type", model.TypeShortExpiry).Error)
	var got2 model.Device
	require.NoError(t, pdb.First(&got2, 1).Error)
	require.Equal(t, model.TypeShortExpiry, got2.Type)
}

func TestDevicePersistence_bytea(t *testing.T) {
	cfg := devicePostgresConfig(t)
	pdb := test.NewPostgresDB(t, cfg)
	defer pdb.Close()

	seedDeviceUser(t, pdb.DB)

	// legacy workaround: devices.type becomes bytea
	require.NoError(t, pdb.Exec(`ALTER TABLE devices ALTER COLUMN type TYPE bytea USING type::bytea`).Error)

	dev := newDevice(1)
	require.NoError(t, pdb.Create(&dev).Error, "insert device into bytea type column")

	var got model.Device
	require.NoError(t, pdb.First(&got, 1).Error)
	require.Equal(t, model.TypeNoExpiry, got.Type)

	require.NoError(t, pdb.Model(&got).Update("type", model.TypeLongExpiry).Error)
	var got2 model.Device
	require.NoError(t, pdb.First(&got2, 1).Error)
	require.Equal(t, model.TypeLongExpiry, got2.Type)
}

// ============================================================
// T007: device lifecycle persistence (file-backed SQLite + PG reconnect)
// ============================================================

func TestDeviceLifecycle_fileSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traggo.db")
	db, err := database.New("sqlite3", path)
	require.NoError(t, err)
	seedDeviceUser(t, db)

	dev := newDevice(1)
	require.NoError(t, db.Create(&dev).Error)

	require.NoError(t, db.Close())
	db2, err := database.New("sqlite3", path)
	require.NoError(t, err, "reopen file-backed sqlite database")
	defer db2.Close()

	var got model.Device
	require.NoError(t, db2.First(&got, 1).Error)
	require.Equal(t, "dev-1", got.Name)
	require.Equal(t, model.TypeNoExpiry, got.Type)

	// read must remain correct after editing the type
	require.NoError(t, db2.Model(&got).Update("type", model.TypeShortExpiry).Error)
	var got2 model.Device
	require.NoError(t, db2.First(&got2, 1).Error)
	require.Equal(t, model.TypeShortExpiry, got2.Type)
}

func TestDeviceLifecycle_pgReconnect(t *testing.T) {
	cfg := devicePostgresConfig(t)

	// use the WP01 harness to create an isolated database, then reopen its DSN
	pdb := test.NewPostgresDB(t, cfg)
	defer pdb.Close()
	seedDeviceUser(t, pdb.DB)

	dev := newDevice(1)
	require.NoError(t, pdb.Create(&dev).Error)

	// Build the DSN for the same disposable database and reopen via database.New
	dsn := deviceDSNForDatabase(pdb.BaseDSN, pdb.DatabaseName)
	gdb2, err := database.New("postgres", dsn)
	require.NoError(t, err, "reopen PostgreSQL database")
	defer gdb2.Close()

	var got model.Device
	require.NoError(t, gdb2.First(&got, 1).Error)
	require.Equal(t, model.TypeNoExpiry, got.Type)
}
