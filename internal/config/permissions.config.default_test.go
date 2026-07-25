package config_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func TestPermissionsConfigDefault(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		role       string
		permission string
		expect     bool
	}{
		{
			name:       "Success/UserCanWriteItems",
			role:       "auth:user",
			permission: config.PermissionItemWrite,
			expect:     true,
		},
		{
			name:       "Success/AnonymousCannotWriteItems",
			role:       "auth:anon",
			permission: config.PermissionItemWrite,
			expect:     false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			role, ok := config.PermissionsConfigDefault.Roles[testCase.role]
			require.True(t, ok)
			require.Equal(
				t,
				testCase.expect,
				slices.Contains(role.Permissions, testCase.permission),
			)
		})
	}
}
