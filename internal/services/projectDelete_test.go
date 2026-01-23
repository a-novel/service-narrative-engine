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

func TestProjectDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000100")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	type projectSelectMock struct {
		resp *dao.Project
		err  error
	}

	type projectDeleteMock struct {
		resp *dao.Project
		err  error
	}

	testCases := []struct {
		name string

		request *services.ProjectDeleteRequest

		projectSelectMock *projectSelectMock
		projectDeleteMock *projectDeleteMock

		expect    *services.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ProjectDeleteRequest{
				ID:     projectID,
				UserID: ownerID,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			projectDeleteMock: &projectDeleteMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangEN,
				Title:     "Test Project",
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
		},
		{
			name: "Error/ProjectSelect/NotFound",

			request: &services.ProjectDeleteRequest{
				ID:     projectID,
				UserID: ownerID,
			},

			projectSelectMock: &projectSelectMock{
				err: dao.ErrProjectSelectNotFound,
			},

			expectErr: dao.ErrProjectSelectNotFound,
		},
		{
			name: "Error/ProjectSelect/Generic",

			request: &services.ProjectDeleteRequest{
				ID:     projectID,
				UserID: ownerID,
			},

			projectSelectMock: &projectSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Forbidden/UserNotOwner",

			request: &services.ProjectDeleteRequest{
				ID:     projectID,
				UserID: otherUserID,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expectErr: services.ErrUserDoesNotOwnProject,
		},
		{
			name: "Error/ProjectDelete",

			request: &services.ProjectDeleteRequest{
				ID:     projectID,
				UserID: ownerID,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			projectDeleteMock: &projectDeleteMock{
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

				projectDeleteRepositorySelect := servicesmocks.NewMockProjectDeleteRepositorySelect(t)
				projectDeleteRepository := servicesmocks.NewMockProjectDeleteRepository(t)

				if testCase.projectSelectMock != nil {
					projectDeleteRepositorySelect.EXPECT().
						Exec(mock.Anything, &dao.ProjectSelectRequest{
							ID: testCase.request.ID,
						}).
						Return(testCase.projectSelectMock.resp, testCase.projectSelectMock.err)
				}

				if testCase.projectDeleteMock != nil {
					projectDeleteRepository.EXPECT().
						Exec(mock.Anything, &dao.ProjectDeleteRequest{
							ID: testCase.request.ID,
						}).
						Return(testCase.projectDeleteMock.resp, testCase.projectDeleteMock.err)
				}

				service := services.NewProjectDelete(projectDeleteRepository, projectDeleteRepositorySelect)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				projectDeleteRepositorySelect.AssertExpectations(t)
				projectDeleteRepository.AssertExpectations(t)
			})
		})
	}
}
