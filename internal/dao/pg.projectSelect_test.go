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

func TestProjectSelect(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.Project

		request *dao.ProjectSelectRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Lang:      "en",
					Title:     "Test Project",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectSelectRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			expect: &dao.Project{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Owner:     uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Lang:      "en",
				Title:     "Test Project",
				Workflow:  []string{},
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectSelectRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			expectErr: dao.ErrProjectSelectNotFound,
		},
	}

	repository := dao.NewProjectSelect()

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
