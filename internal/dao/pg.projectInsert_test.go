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

func TestProjectInsert(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.Project

		request *dao.ProjectInsertRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectInsertRequest{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Lang:     "en",
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
				Now:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Project{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Lang:      "en",
				Title:     "Test Project",
				Workflow:  []string{"agora:idea@v1.0.0"},
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/AlreadyExists",

			fixtures: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Lang:      "en",
					Title:     "Existing Project",
					Workflow:  []string{"agora:concept@v2.0.0"},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectInsertRequest{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Owner:    uuid.MustParse("00000000-0000-0000-0000-000000000200"),
				Lang:     "fr",
				Title:    "Duplicate Project",
				Workflow: []string{"agora:idea@v1.0.0"},
				Now:      time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrProjectInsertAlreadyExists,
		},
	}

	repository := dao.NewProjectInsert()

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

				res, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, res)
			})
		})
	}
}
