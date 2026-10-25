package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	dType := DeviceType("abc")
	require.Error(t, dType.Valid(), "unknown type")

	require.Equal(t, TypeShortExpiry.Seconds(), 3600)
	require.Equal(t, TypeLongExpiry.Seconds(), 2592000)
	require.Equal(t, TypeNoExpiry.Seconds(), 31536000)
	value, err := TypeNoExpiry.Value()
	require.NoError(t, err)
	require.Equal(t, value, "NoExpiry")
	require.Nil(t, TypeNoExpiry.Valid())

	var scan DeviceType

	err = scan.Scan([]byte("NoExpiry"))
	require.NoError(t, err)
	require.Equal(t, TypeNoExpiry, scan)
}

func TestDeviceType_Scan(t *testing.T) {
	t.Run("accepts string (PostgreSQL text / SQLite)", func(t *testing.T) {
		var dt DeviceType
		require.NoError(t, dt.Scan("NoExpiry"))
		require.Equal(t, DeviceType("NoExpiry"), dt)
	})

	t.Run("accepts byte slice (PostgreSQL legacy bytea)", func(t *testing.T) {
		var dt DeviceType
		require.NoError(t, dt.Scan([]byte("ShortExpiry")))
		require.Equal(t, DeviceType("ShortExpiry"), dt)
	})

	t.Run("nil returns defined error", func(t *testing.T) {
		var dt DeviceType
		err := dt.Scan(nil)
		require.ErrorIs(t, err, ErrDeviceTypeNil)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		var dt DeviceType
		err := dt.Scan(3.14)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported value type")
		require.Contains(t, err.Error(), "expected string or []byte")
	})

	t.Run("value round trips to a string", func(t *testing.T) {
		for _, v := range []DeviceType{TypeShortExpiry, TypeLongExpiry, TypeNoExpiry} {
			val, err := v.Value()
			require.NoError(t, err)
			require.IsType(t, "", val)
			var back DeviceType
			require.NoError(t, back.Scan(val))
			require.Equal(t, v, back)
		}
	})

	t.Run("scan does not mutate receiver on error", func(t *testing.T) {
		var dt DeviceType = TypeNoExpiry
		require.Error(t, dt.Scan(42))
		require.Equal(t, TypeNoExpiry, dt)
	})
}
