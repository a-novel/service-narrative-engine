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

func TestModuleListIDs(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.Module

		request *dao.ModuleListIDsRequest

		expect    []string
		expectErr error
	}{
		{
			name: "Success/ListAll",

			fixtures: []*dao.Module{
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Generate ideas",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "concept",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Develop concepts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "draft",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Write drafts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
			},

			expect: []string{"concept", "draft", "idea"},
		},
		{
			name: "Success/WithLimit",

			fixtures: []*dao.Module{
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Generate ideas",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "concept",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Develop concepts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "draft",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Write drafts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     2,
			},

			expect: []string{"concept", "draft"},
		},
		{
			name: "Success/WithOffset",

			fixtures: []*dao.Module{
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Generate ideas",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "concept",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Develop concepts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "draft",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Write drafts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
				Offset:    1,
			},

			expect: []string{"draft", "idea"},
		},
		{
			name: "Success/LimitAndOffset",

			fixtures: []*dao.Module{
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Generate ideas",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "concept",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Develop concepts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "draft",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Write drafts",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "outline",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Create outlines",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     2,
				Offset:    1,
			},

			expect: []string{"draft", "idea"},
		},
		{
			name: "Success/Deduplicates",

			fixtures: []*dao.Module{
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Idea v1",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "2.0.0",
					Preversion:  "",
					Description: "Idea v2",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "concept",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Concept v1",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
			},

			expect: []string{"concept", "idea"},
		},
		{
			name: "Success/FiltersByNamespace",

			fixtures: []*dao.Module{
				{
					ID:          "idea",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Agora idea",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "concept",
					Namespace:   "agora",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Agora concept",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:          "chapter",
					Namespace:   "scribe",
					Version:     "1.0.0",
					Preversion:  "",
					Description: "Scribe chapter",
					Schema:      testSchema,
					CreatedAt:   time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
			},

			expect: []string{"concept", "idea"},
		},
		{
			name: "Success/Empty",

			request: &dao.ModuleListIDsRequest{
				Namespace: "agora",
			},

			expect: []string{},
		},
	}

	repository := dao.NewModuleListIDs()

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

				ids, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, ids)
			})
		})
	}
}
