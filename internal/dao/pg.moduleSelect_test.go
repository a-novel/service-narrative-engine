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

func TestModuleSelect(t *testing.T) {
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

		request *dao.ModuleSelectRequest

		expect    *dao.Module
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
					Description: "A test module",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: "",
			},

			expect: &dao.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "",
				Description: "A test module",
				Schema:      testSchema,
				UI:          testUi,
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/MultipleVersions",

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
			},

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "2.0.0",
				Preversion: "",
			},

			expect: &dao.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "2.0.0",
				Preversion:  "",
				Description: "Version 2",
				Schema:      testSchema,
				UI:          testUi,
				CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: "",
			},

			expectErr: dao.ErrModuleSelectNotFound,
		},
		{
			name: "Success/EmptyVersionReturnsLatestStable",

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

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "",
				Preversion: "",
			},

			expect: &dao.Module{
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
		{
			name: "Success/EmptyVersionIgnoresPreversions",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Stable version",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "2.0.0",
					Preversion:  "-beta-1",
					Description: "Beta version",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "3.0.0",
					Preversion:  "-rc-1",
					Description: "RC version",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "",
				Preversion: "",
			},

			expect: &dao.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "",
				Description: "Stable version",
				Schema:      testSchema,
				UI:          testUi,
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/SelectWithPreversion",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Stable version",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-1",
					Description: "Beta version",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: "-beta-1",
			},

			expect: &dao.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "-beta-1",
				Description: "Beta version",
				Schema:      testSchema,
				UI:          testUi,
				CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/WrongVersion",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Test module",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "2.0.0",
				Preversion: "",
			},

			expectErr: dao.ErrModuleSelectNotFound,
		},
		{
			name: "Error/WrongPreversion",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Test module",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleSelectRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: "-beta-1",
			},

			expectErr: dao.ErrModuleSelectNotFound,
		},
	}

	repository := dao.NewModuleSelect()

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

				module, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, module)
			})
		})
	}
}
