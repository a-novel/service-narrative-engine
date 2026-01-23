package dao_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models"
)

func TestModuleDelete(t *testing.T) {
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

		request *dao.ModuleDeleteRequest

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

			request: &dao.ModuleDeleteRequest{
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
			name: "Success/WithPreversion",

			fixtures: []*dao.Module{
				{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-1",
					Description: "A beta test module",
					Schema:      testSchema,
					UI:          testUi,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleDeleteRequest{
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
				Description: "A beta test module",
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

			request: &dao.ModuleDeleteRequest{
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
				Description: "Version 1",
				Schema:      testSchema,
				UI:          testUi,
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.ModuleDeleteRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: "",
			},

			expectErr: dao.ErrModuleDeleteNotFound,
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

			request: &dao.ModuleDeleteRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "2.0.0",
				Preversion: "",
			},

			expectErr: dao.ErrModuleDeleteNotFound,
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

			request: &dao.ModuleDeleteRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Version:    "1.0.0",
				Preversion: "-beta-1",
			},

			expectErr: dao.ErrModuleDeleteNotFound,
		},
	}

	repository := dao.NewModuleDelete()

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

				// Verify deletion by attempting to select the deleted module
				if testCase.expectErr == nil {
					deletedModule := new(dao.Module)
					err = db.NewSelect().Model(deletedModule).
						Where("id = ? AND namespace = ? AND version = ?",
							testCase.request.ID, testCase.request.Namespace, testCase.request.Version).
						Scan(ctx)
					require.ErrorIs(t, err, sql.ErrNoRows)
				}
			})
		})
	}
}
