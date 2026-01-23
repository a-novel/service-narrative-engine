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

func TestProjectSelect(t *testing.T) {
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

	testCases := []struct {
		name string

		request *services.ProjectSelectRequest

		projectSelectMock *projectSelectMock

		expect    *services.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ProjectSelectRequest{
				ID:     projectID,
				UserID: ownerID,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-namespace:test-module@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangEN,
				Title:     "Test Project",
				Workflow:  []string{"test-namespace:test-module@v1.0.0"},
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
		},
		{
			name: "Error/InvalidRequest/MissingID",

			request: &services.ProjectSelectRequest{
				UserID: ownerID,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingUserID",

			request: &services.ProjectSelectRequest{
				ID: projectID,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectSelect/NotFound",

			request: &services.ProjectSelectRequest{
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

			request: &services.ProjectSelectRequest{
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

			request: &services.ProjectSelectRequest{
				ID:     projectID,
				UserID: otherUserID,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-namespace:test-module@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expectErr: services.ErrUserDoesNotOwnProject,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				projectSelectRepository := servicesmocks.NewMockProjectSelectRepository(t)

				if testCase.projectSelectMock != nil {
					projectSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.ProjectSelectRequest{
							ID: testCase.request.ID,
						}).
						Return(testCase.projectSelectMock.resp, testCase.projectSelectMock.err)
				}

				service := services.NewProjectSelect(projectSelectRepository)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				projectSelectRepository.AssertExpectations(t)
			})
		})
	}
}
