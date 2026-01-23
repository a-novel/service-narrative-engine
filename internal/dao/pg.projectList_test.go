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

func TestProjectList(t *testing.T) {
	owner1 := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	owner2 := uuid.MustParse("00000000-0000-0000-0000-000000000200")

	testCases := []struct {
		name string

		fixtures []*dao.Project

		request *dao.ProjectListRequest

		expect    []*dao.Project
		expectErr error
	}{
		{
			name: "Success/ListAll",

			fixtures: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 1",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 2",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Owner:     owner2,
					Lang:      "fr",
					Title:     "Project 3",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectListRequest{
				Owner: owner1,
			},

			expect: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 2",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 1",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/WithLimit",

			fixtures: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 1",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 2",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectListRequest{
				Owner: owner1,
				Limit: 1,
			},

			expect: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 2",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/WithOffset",

			fixtures: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 1",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 2",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.ProjectListRequest{
				Owner:  owner1,
				Offset: 1,
			},

			expect: []*dao.Project{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Owner:     owner1,
					Lang:      "en",
					Title:     "Project 1",
					Workflow:  []string{},
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Success/EmptyResult",

			request: &dao.ProjectListRequest{
				Owner: owner1,
			},

			expect: []*dao.Project{},
		},
	}

	repository := dao.NewProjectList()

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
