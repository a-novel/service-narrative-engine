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

func TestSchemaMetaList(t *testing.T) {
	testData1 := map[string]any{
		"title": "Test Story 1",
	}
	testData2 := map[string]any{
		"title": "Test Story 2",
	}

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000001000")

	testCases := []struct {
		name string

		fixtures []*dao.Schema

		request *dao.SchemaMetaListRequest

		expect    []*dao.SchemaMeta
		expectErr error
	}{
		{
			name: "Success/ListLatestForEachModule",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
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
					ModuleID:         "module-a",
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
					ModuleID:         "module-b",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceFork,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaMetaListRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
			},

			expect: []*dao.SchemaMeta{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-b",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceFork,
					IsLatest:         true,
					IsNil:            false,
					CreatedAt:        time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "2.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					IsLatest:         true,
					IsNil:            false,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/ExcludesNilData",

			fixtures: []*dao.Schema{
				{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:           &ownerID,
					ModuleID:        "module-a",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            testData1,
					CreatedAt:       time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:           &ownerID,
					ModuleID:        "module-a",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "2.0.0",
					Source:          dao.SchemaSourceAI,
					Data:            nil,
					CreatedAt:       time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:           &ownerID,
					ModuleID:        "module-b",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceFork,
					Data:            testData1,
					CreatedAt:       time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaMetaListRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
			},

			expect: []*dao.SchemaMeta{
				{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					ProjectID:       uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:           &ownerID,
					ModuleID:        "module-b",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceFork,
					IsLatest:        true,
					IsNil:           false,
					CreatedAt:       time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/DifferentNamespaces",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "namespace-1",
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
					ModuleID:         "module-a",
					ModuleNamespace:  "namespace-2",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData2,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaMetaListRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
			},

			expect: []*dao.SchemaMeta{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "namespace-2",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					IsLatest:         true,
					IsNil:            false,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "namespace-1",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					IsLatest:         true,
					IsNil:            false,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/FiltersByProject",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000200"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceAI,
					Data:             testData2,
					CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaMetaListRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
			},

			expect: []*dao.SchemaMeta{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					IsLatest:         true,
					IsNil:            false,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/EmptyResult",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "module-a",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             testData1,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaMetaListRequest{
				ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000200"),
			},

			expect: []*dao.SchemaMeta(nil),
		},
	}

	repository := dao.NewSchemaMetaList()

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

				schemas, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, schemas)
			})
		})
	}
}
