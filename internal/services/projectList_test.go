package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestProjectList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	projectID1 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID2 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	type projectListMock struct {
		resp []*dao.Project
		err  error
	}

	testCases := []struct {
		name string

		request *services.ProjectListRequest

		projectListMock *projectListMock

		expect    []*services.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: 0,
			},

			projectListMock: &projectListMock{
				resp: []*dao.Project{
					{
						ID:        projectID1,
						Owner:     userID,
						Lang:      "en",
						Title:     "Test Project",
						Workflow:  []string{"module1", "module2"},
						CreatedAt: baseTime,
						UpdatedAt: baseTime,
					},
				},
			},

			expect: []*services.Project{
				{
					ID:        projectID1,
					Owner:     userID,
					Lang:      "en",
					Title:     "Test Project",
					Workflow:  []string{"module1", "module2"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},
		},
		{
			name: "Success/MultipleProjects",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: 0,
			},

			projectListMock: &projectListMock{
				resp: []*dao.Project{
					{
						ID:        projectID1,
						Owner:     userID,
						Lang:      "en",
						Title:     "Test Project 1",
						Workflow:  []string{"module1"},
						CreatedAt: baseTime,
						UpdatedAt: baseTime,
					},
					{
						ID:        projectID2,
						Owner:     userID,
						Lang:      "fr",
						Title:     "Test Project 2",
						Workflow:  []string{"module2", "module3"},
						CreatedAt: baseTime.Add(time.Hour),
						UpdatedAt: baseTime.Add(time.Hour),
					},
				},
			},

			expect: []*services.Project{
				{
					ID:        projectID1,
					Owner:     userID,
					Lang:      "en",
					Title:     "Test Project 1",
					Workflow:  []string{"module1"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
				{
					ID:        projectID2,
					Owner:     userID,
					Lang:      "fr",
					Title:     "Test Project 2",
					Workflow:  []string{"module2", "module3"},
					CreatedAt: baseTime.Add(time.Hour),
					UpdatedAt: baseTime.Add(time.Hour),
				},
			},
		},
		{
			name: "Success/WithOffset",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: 5,
			},

			projectListMock: &projectListMock{
				resp: []*dao.Project{
					{
						ID:        projectID1,
						Owner:     userID,
						Lang:      "en",
						Title:     "Test Project",
						Workflow:  []string{"module1"},
						CreatedAt: baseTime,
						UpdatedAt: baseTime,
					},
				},
			},

			expect: []*services.Project{
				{
					ID:        projectID1,
					Owner:     userID,
					Lang:      "en",
					Title:     "Test Project",
					Workflow:  []string{"module1"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},
		},
		{
			name: "Success/EmptyResult",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: 0,
			},

			projectListMock: &projectListMock{
				resp: []*dao.Project{},
			},

			expect: []*services.Project{},
		},
		{
			name: "Error/InvalidRequest/MissingUserID",

			request: &services.ProjectListRequest{
				UserID: uuid.Nil,
				Limit:  10,
				Offset: 0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingLimit",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  0,
				Offset: 0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/LimitTooLarge",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  129,
				Offset: 0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetNegative",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: -1,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetTooLarge",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: 8193,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/RepositoryError",

			request: &services.ProjectListRequest{
				UserID: userID,
				Limit:  10,
				Offset: 0,
			},

			projectListMock: &projectListMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				projectListRepository := servicesmocks.NewMockProjectListRepository(t)

				if testCase.projectListMock != nil {
					projectListRepository.EXPECT().
						Exec(mock.Anything, &dao.ProjectListRequest{
							Owner:  testCase.request.UserID,
							Limit:  testCase.request.Limit,
							Offset: testCase.request.Offset,
						}).
						Return(testCase.projectListMock.resp, testCase.projectListMock.err)
				}

				service := services.NewProjectList(
					projectListRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				projectListRepository.AssertExpectations(t)
			})
		})
	}
}
