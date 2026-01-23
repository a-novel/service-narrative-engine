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

func TestProjectDelete(t *testing.T) {
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000100")

	testData := map[string]any{
		"title": "Test Story",
	}

	testCases := []struct {
		name string

		fixtures       []*dao.Project
		schemaFixtures []*dao.Schema

		request *dao.ProjectDeleteRequest

		expect               *dao.Project
		expectErr            error
		expectSchemasDeleted bool
	}{
		{
			name: "Success",

			fixtures: []*dao.Project{
				{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      "en",
					Title:     "Test Project",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectDeleteRequest{
				ID: projectID,
			},

			expect: &dao.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      "en",
				Title:     "Test Project",
				Workflow:  []string{},
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/WithSchemas",

			fixtures: []*dao.Project{
				{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      "en",
					Title:     "Test Project",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			schemaFixtures: []*dao.Schema{
				{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000010"),
					ProjectID:       projectID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            testData,
					CreatedAt:       time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:              uuid.MustParse("00000000-0000-0000-0000-000000000011"),
					ProjectID:       projectID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceAI,
					Data:            testData,
					CreatedAt:       time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectDeleteRequest{
				ID: projectID,
			},

			expect: &dao.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      "en",
				Title:     "Test Project",
				Workflow:  []string{},
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectSchemasDeleted: true,
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectDeleteRequest{
				ID: projectID,
			},

			expectErr: dao.ErrProjectDeleteNotFound,
		},
	}

	repository := dao.NewProjectDelete()

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

				if len(testCase.schemaFixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.schemaFixtures).Exec(ctx)
					require.NoError(t, err)
				}

				res, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, res)

				if testCase.expectErr == nil {
					// Verify schemas were deleted if expected
					if testCase.expectSchemasDeleted {
						var schemas []*dao.Schema

						err = db.NewSelect().
							Model(&schemas).
							Where("project_id = ?", testCase.request.ID).
							Scan(ctx)
						require.NoError(t, err)
						require.Empty(t, schemas)
					}
				}
			})
		})
	}
}
