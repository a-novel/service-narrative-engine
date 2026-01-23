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

func TestSchemaListVersions(t *testing.T) {
	testData := map[string]any{
		"title": "Test Story",
	}

	testCases := []struct {
		name string

		fixtures []*dao.Schema

		request *dao.SchemaListVersionsRequest

		expect    []*dao.SchemaVersion
		expectErr error
	}{
		{
			name: "Success/ListAll",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "2.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "3.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceFork,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaListVersionsRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           0,
				Offset:          0,
			},

			expect: []*dao.SchemaVersion{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/WithLimit",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "2.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "3.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceFork,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaListVersionsRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           2,
				Offset:          0,
			},

			expect: []*dao.SchemaVersion{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/WithOffset",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "2.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "3.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceFork,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaListVersionsRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           0,
				Offset:          1,
			},

			expect: []*dao.SchemaVersion{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/FilterByModule",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "other-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceExternal,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaListVersionsRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           0,
				Offset:          0,
			},

			expect: []*dao.SchemaVersion{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/EmptyResult",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaListVersionsRequest{
				ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000200"),
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           0,
				Offset:          0,
			},

			expect: []*dao.SchemaVersion{},
		},
	}

	repository := dao.NewSchemaListVersions()

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
