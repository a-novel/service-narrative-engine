package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestModuleListNamespaces(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.Module

		request *dao.ModuleListNamespacesRequest

		expect    []string
		expectErr error
	}{
		{
			name: "Success/ListAll",

			fixtures: []*dao.Module{
				{
					ID:          "module-1",
					Namespace:   "alpha",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in alpha",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-2",
					Namespace:   "beta",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in beta",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-3",
					Namespace:   "gamma",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in gamma",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListNamespacesRequest{},

			expect: []string{"alpha", "beta", "gamma"},
		},
		{
			name: "Success/WithLimit",

			fixtures: []*dao.Module{
				{
					ID:          "module-1",
					Namespace:   "alpha",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in alpha",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-2",
					Namespace:   "beta",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in beta",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-3",
					Namespace:   "gamma",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in gamma",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListNamespacesRequest{
				Limit: 2,
			},

			expect: []string{"alpha", "beta"},
		},
		{
			name: "Success/WithOffset",

			fixtures: []*dao.Module{
				{
					ID:          "module-1",
					Namespace:   "alpha",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in alpha",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-2",
					Namespace:   "beta",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in beta",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-3",
					Namespace:   "gamma",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in gamma",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListNamespacesRequest{
				Offset: 1,
			},

			expect: []string{"beta", "gamma"},
		},
		{
			name: "Success/LimitAndOffset",

			fixtures: []*dao.Module{
				{
					ID:          "module-1",
					Namespace:   "alpha",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in alpha",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-2",
					Namespace:   "beta",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in beta",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-3",
					Namespace:   "gamma",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in gamma",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-4",
					Namespace:   "delta",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in delta",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListNamespacesRequest{
				Limit:  2,
				Offset: 1,
			},

			expect: []string{"beta", "delta"},
		},
		{
			name: "Success/Deduplicates",

			fixtures: []*dao.Module{
				{
					ID:          "module-1",
					Namespace:   "alpha",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "First module in alpha",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-2",
					Namespace:   "alpha",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Second module in alpha",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "module-3",
					Namespace:   "beta",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Module in beta",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListNamespacesRequest{},

			expect: []string{"alpha", "beta"},
		},
		{
			name: "Success/Empty",

			request: &dao.ModuleListNamespacesRequest{},

			expect: []string{},
		},
	}

	repository := dao.NewModuleListNamespaces()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				namespaces, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, namespaces)
			})
		})
	}
}
