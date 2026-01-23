package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models"
)

func TestModuleListVersions(t *testing.T) {
	testSchema := jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"field1": {
				Type: "string",
			},
		},
		Required: []string{"field1"},
	}

	testUi := models.ModuleUi{
		Component: "input",
		Params: models.ModuleUiParams{
			"placeholder": "Enter value",
		},
		Target: "field1",
	}

	testCases := []struct {
		name string

		fixtures []*dao.Module

		request *dao.ModuleListVersionsRequest

		expect    []*dao.ModuleVersion
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Version 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "3.0.0",
					Preversion:  "",
					Description: "Version 3",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
			},

			expect: []*dao.ModuleVersion{
				{Version: "3.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)},
				{Version: "2.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/PreversionsFilteredByDefault",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Stable 1.0.0",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-1",
					Description: "Beta 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-2",
					Description: "Beta 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "0.9.0",
					Preversion:  "",
					Description: "Stable 0.9.0",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
			},

			// Preversions are filtered out by default
			expect: []*dao.ModuleVersion{
				{Version: "1.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
				{Version: "0.9.0", Preversion: "", CreatedAt: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/WithPreversions",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Stable 1.0.0",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-1",
					Description: "Beta 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-2",
					Description: "Beta 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "0.9.0",
					Preversion:  "",
					Description: "Stable 0.9.0",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Preversion: true,
			},

			// Sorted by version DESC, then empty preversion first, then preversion by created_at DESC
			expect: []*dao.ModuleVersion{
				{Version: "1.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "-beta-2", CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "-beta-1", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
				{Version: "0.9.0", Preversion: "", CreatedAt: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/Limit",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Version 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "3.0.0",
					Preversion:  "",
					Description: "Version 3",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     2,
			},

			expect: []*dao.ModuleVersion{
				{Version: "3.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)},
				{Version: "2.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/Offset",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Version 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "3.0.0",
					Preversion:  "",
					Description: "Version 3",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Offset:    1,
			},

			expect: []*dao.ModuleVersion{
				{Version: "2.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/LimitAndOffset",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Version 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "3.0.0",
					Preversion:  "",
					Description: "Version 3",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "4.0.0",
					Preversion:  "",
					Description: "Version 4",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     2,
				Offset:    1,
			},

			expect: []*dao.ModuleVersion{
				{Version: "3.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)},
				{Version: "2.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/FilterByNamespace",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "namespace-1",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "NS1 Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "namespace-2",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "NS2 Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "namespace-1",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "NS1 Version 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "namespace-1",
			},

			expect: []*dao.ModuleVersion{
				{Version: "2.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/FilterByVersion",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Version 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Version 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "3.0.0",
					Preversion:  "",
					Description: "Version 3",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Version:   "2.0.0",
			},

			expect: []*dao.ModuleVersion{
				{Version: "2.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/FilterByVersionWithPreversions",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Stable 1.0.0",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-1",
					Description: "Beta 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-2",
					Description: "Beta 2",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Stable 2.0.0",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "-alpha-1",
					Description: "Alpha 1",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListVersionsRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: true,
			},

			// Only version 1.0.0 modules (stable first, then preversions by created_at DESC)
			expect: []*dao.ModuleVersion{
				{Version: "1.0.0", Preversion: "", CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "-beta-2", CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)},
				{Version: "1.0.0", Preversion: "-beta-1", CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)},
			},
		},
		{
			name: "Success/Empty",

			request: &dao.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
			},

			expect: []*dao.ModuleVersion{},
		},
	}

	repository := dao.NewModuleListVersions()

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

				versions, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, versions)
			})
		})
	}
}
