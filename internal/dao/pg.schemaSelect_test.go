package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestSchemaSelect(t *testing.T) {
	testData1 := map[string]any{
		"title": "Test Story 1",
	}
	testData2 := map[string]any{
		"title": "Test Story 2",
	}
	testData3 := map[string]any{
		"title": "Test Story 3",
	}

	id1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000001000")

	testCases := []struct {
		name string

		fixtures []*dao.Schema

		request *dao.SchemaSelectRequest

		expect    *dao.Schema
		expectErr error
	}{
		{
			name: "Success/ByID",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "2.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData2,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaSelectRequest{
				ID: &id1,
			},

			expect: &dao.Schema{
				ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "1.0.0",
				ModulePreversion: "",
				Source:           dao.SchemaSourceUser,
				Data:             testData1,
				CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/LatestByProjectAndModule",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "2.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData2,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "other-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceFork,
					Data:             testData3,
					CreatedAt:        time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaSelectRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
			},

			expect: &dao.Schema{
				ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "2.0.0",
				ModulePreversion: "",
				Source:           dao.SchemaSourceAI,
				Data:             testData2,
				CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/NotFoundByID",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceExternal,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaSelectRequest{
				ID: &id2,
			},

			expectErr: dao.ErrSchemaSelectNotFound,
		},
		{
			name: "Error/NotFoundByProject",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaSelectRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000200"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
			},

			expectErr: dao.ErrSchemaSelectNotFound,
		},
	}

	repository := dao.NewSchemaGet()

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

				schema, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, schema)
			})
		})
	}
}
