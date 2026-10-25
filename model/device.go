package model

import (
	"database/sql/driver"
	"errors"
	"time"
)

// ErrDeviceTypeNil is returned when a NULL device type is scanned.
var ErrDeviceTypeNil = errors.New("device type: cannot scan NULL (nil value)")

// Device represents something which can connect to traggo
type Device struct {
	ID        int    `gorm:"primary_key;unique_index;AUTO_INCREMENT"`
	Token     string `gorm:"unique"`
	Name      string
	UserID    int `gorm:"type:int REFERENCES users(id) ON DELETE CASCADE"`
	CreatedAt time.Time
	Type      DeviceType
	ActiveAt  time.Time
}

// DeviceType the device type
type DeviceType string

// Value for db
func (t DeviceType) Value() (driver.Value, error) {
	return string(t), nil
}

// Scan for db
func (t *DeviceType) Scan(value interface{}) error {
	if value == nil {
		return ErrDeviceTypeNil
	}
	s := (*string)(t)
	return scanStringEnum("device type", s, value)
}

// Seconds returns the amount of seconds after the device expires.
func (t DeviceType) Seconds() int {
	switch t {
	case TypeLongExpiry:
		return 60 * 60 * 24 * 30
	case TypeShortExpiry:
		return 60 * 60
	case TypeNoExpiry:
		return 60 * 60 * 24 * 365
	default:
		return 0
	}
}

// Valid checks if the device type is valid.
func (t DeviceType) Valid() error {
	switch t {
	case TypeNoExpiry, TypeShortExpiry, TypeLongExpiry:
		return nil
	default:
		return errors.New("unknown device type")
	}
}

// Device types
const (
	TypeShortExpiry DeviceType = "ShortExpiry"
	TypeLongExpiry  DeviceType = "LongExpiry"
	TypeNoExpiry    DeviceType = "NoExpiry"
)
