package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"

	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func TestNewPostgresPreset(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		connection postgrespresets.Connection

		expectAddr       string
		expectUser       string
		expectPassword   string
		expectDatabase   string
		expectTLSEnabled bool
		expectPanic      string
	}{
		{
			name: "Success/Discrete",
			connection: postgrespresets.Connection{
				DSN:        "not-used",
				Host:       "postgres.internal",
				Port:       5433,
				User:       "template",
				Password:   "password-with-:/@%",
				Database:   "template",
				TLSEnabled: false,
			},
			expectAddr:       "postgres.internal:5433",
			expectUser:       "template",
			expectPassword:   "password-with-:/@%",
			expectDatabase:   "template",
			expectTLSEnabled: false,
		},
		{
			name: "Success/DiscreteTLS",
			connection: postgrespresets.Connection{
				Host:       "postgres.internal",
				Port:       5432,
				User:       "template",
				Password:   "password",
				Database:   "template",
				TLSEnabled: true,
			},
			expectAddr:       "postgres.internal:5432",
			expectUser:       "template",
			expectPassword:   "password",
			expectDatabase:   "template",
			expectTLSEnabled: true,
		},
		{
			name: "Success/LegacyDSN",
			connection: postgrespresets.Connection{
				DSN: "postgres://legacy-user@legacy.internal:6432/legacy-db?sslmode=disable",
			},
			expectAddr:       "legacy.internal:6432",
			expectUser:       "legacy-user",
			expectPassword:   "",
			expectDatabase:   "legacy-db",
			expectTLSEnabled: false,
		},
		{
			name: "Error/MissingDiscretePassword",
			connection: postgrespresets.Connection{
				Host:     "postgres.internal",
				Port:     5432,
				User:     "template",
				Database: "template",
			},
			expectPanic: "postgres password is empty",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.expectPanic != "" {
				require.PanicsWithValue(t, testCase.expectPanic, func() {
					config.NewPostgresPreset(testCase.connection, 12, 8)
				})

				return
			}

			preset := config.NewPostgresPreset(testCase.connection, 12, 8)
			driverConfig := pgdriver.NewConnector(preset.Options()...).Config()

			require.Equal(t, testCase.expectAddr, driverConfig.Addr)
			require.Equal(t, testCase.expectUser, driverConfig.User)
			require.Equal(t, testCase.expectPassword, driverConfig.Password)
			require.Equal(t, testCase.expectDatabase, driverConfig.Database)
			require.Equal(t, testCase.expectTLSEnabled, driverConfig.TLSConfig != nil)
			require.Equal(t, 3*time.Minute, driverConfig.DialTimeout)
			require.Equal(t, 12, preset.MaxOpenConns)
			require.Equal(t, 8, preset.MaxIdleConns)
		})
	}
}
