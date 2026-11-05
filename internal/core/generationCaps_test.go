package core_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
)

func TestGenerationCapsPolicy(t *testing.T) {
	t.Parallel()

	// A premium user inherits the user's role, and with it the right to write projects.
	permissions := serviceauthentication.Permissions{Roles: map[string]serviceauthentication.Role{
		"auth:anon": {Priority: 0},
		"auth:user": {
			Priority: 1, Inherits: []string{"auth:anon"},
			Permissions: []string{config.PermissionProjectRead, config.PermissionProjectWrite},
		},
		"premium": {Priority: 2, Inherits: []string{"auth:user"}},
	}}

	userCaps := config.RoleCaps{
		Concurrency: lo.ToPtr[int32](2),
		Windows: map[time.Duration]config.CapsWindow{
			720 * time.Hour: {InputTokens: lo.ToPtr[int64](50_000_000)},
			5 * time.Hour:   {InputTokens: lo.ToPtr[int64](2_000_000), OutputTokens: lo.ToPtr[int64](200_000)},
		},
	}
	premiumCaps := config.RoleCaps{Concurrency: lo.ToPtr[int32](4)}

	// The windows come out shortest first.
	expectUserCaps := &core.GenerationCaps{
		Concurrency: lo.ToPtr[int32](2),
		Windows: []*core.GenerationCapWindow{
			{Duration: 5 * time.Hour, InputTokens: lo.ToPtr[int64](2_000_000), OutputTokens: lo.ToPtr[int64](200_000)},
			{Duration: 720 * time.Hour, InputTokens: lo.ToPtr[int64](50_000_000)},
		},
	}

	testCases := []struct {
		name string

		permissions serviceauthentication.Permissions
		caps        config.Caps
		roles       []string

		expect    *core.GenerationCaps
		expectErr error
	}{
		{
			name: "Success",

			permissions: permissions,
			caps:        config.Caps{Roles: map[string]config.RoleCaps{"auth:user": userCaps}},
			roles:       []string{"auth:user"},

			expect: expectUserCaps,
		},
		{
			// A role the caps do not name takes the caps of a role it inherits.
			name: "Success/Inherited",

			permissions: permissions,
			caps:        config.Caps{Roles: map[string]config.RoleCaps{"auth:user": userCaps}},
			roles:       []string{"premium"},

			expect: expectUserCaps,
		},
		{
			name: "Success/HighestPriority",

			permissions: permissions,
			caps: config.Caps{Roles: map[string]config.RoleCaps{
				"auth:user": userCaps,
				"premium":   premiumCaps,
			}},
			roles: []string{"auth:user", "premium"},

			expect: &core.GenerationCaps{Concurrency: lo.ToPtr[int32](4)},
		},
		{
			name: "Success/Defaults",

			permissions: config.PermissionsConfigDefault,
			caps:        config.CapsConfigDefault,
			roles:       []string{"auth:user"},

			expect: &core.GenerationCaps{
				Concurrency: lo.ToPtr[int32](2),
				Windows: []*core.GenerationCapWindow{
					{
						Duration:    5 * time.Hour,
						InputTokens: lo.ToPtr[int64](2_000_000), OutputTokens: lo.ToPtr[int64](200_000),
					},
					{
						Duration:    720 * time.Hour,
						InputTokens: lo.ToPtr[int64](50_000_000), OutputTokens: lo.ToPtr[int64](5_000_000),
					},
				},
			},
		},
		{
			// An admin's empty caps limit nothing, and outrank the user's caps the admin inherits.
			name: "Success/Uncapped",

			permissions: config.PermissionsConfigDefault,
			caps:        config.CapsConfigDefault,
			roles:       []string{"auth:user", "auth:admin"},

			expect: &core.GenerationCaps{},
		},
		{
			// A superadmin inherits the admin's permission.
			name: "Success/InheritedUncapped",

			permissions: config.PermissionsConfigDefault,
			caps:        config.CapsConfigDefault,
			roles:       []string{"auth:superadmin"},

			expect: &core.GenerationCaps{},
		},
		{
			name: "Error/NoCaps",

			permissions: config.PermissionsConfigDefault,
			caps:        config.CapsConfigDefault,
			roles:       []string{"auth:anon"},

			expectErr: core.ErrNoGenerationCaps,
		},
		{
			name: "Error/UnknownRole",

			permissions: config.PermissionsConfigDefault,
			caps:        config.CapsConfigDefault,
			roles:       []string{"auth:visitor"},

			expectErr: core.ErrNoGenerationCaps,
		},
		{
			// Every role that writes projects takes caps.
			name: "Error/WriterWithoutCaps",

			permissions: permissions,
			caps:        config.Caps{Roles: map[string]config.RoleCaps{"premium": premiumCaps}},

			expectErr: core.ErrInvalidGenerationCaps,
		},
		{
			name: "Error/CapsOfUnknownRole",

			permissions: permissions,
			caps: config.Caps{Roles: map[string]config.RoleCaps{
				"auth:user":    userCaps,
				"auth:visitor": premiumCaps,
			}},

			expectErr: core.ErrInvalidGenerationCaps,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			policy, err := core.NewGenerationCapsPolicy(testCase.permissions, testCase.caps)
			if err != nil {
				require.ErrorIs(t, err, testCase.expectErr)

				return
			}

			caps, err := policy.Resolve(testCase.roles)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, caps)
		})
	}
}
