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

func TestProjectUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000100")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedTime := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)

	type projectSelectMock struct {
		resp *dao.Project
		err  error
	}

	type projectUpdateMock struct {
		resp *dao.Project
		err  error
	}

	type schemaInsertMock struct {
		resp *dao.Schema
		err  error
	}

	type moduleSelectMock struct {
		resp *dao.Module
		err  error
	}

	testCases := []struct {
		name string

		request *services.ProjectUpdateRequest

		projectSelectMock  *projectSelectMock
		moduleSelectMocks  []moduleSelectMock
		expectModuleSelect int
		projectUpdateMock  *projectUpdateMock
		schemaInsertMocks  []schemaInsertMock

		expect    *services.Project
		expectErr error
	}{
		{
			name: "Success/TitleOnlyUpdate",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Updated Title",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "1.0.0"}},
			},
			expectModuleSelect: 1,

			projectUpdateMock: &projectUpdateMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Updated Title",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: updatedTime,
				},
			},

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangEN,
				Title:     "Updated Title",
				Workflow:  []string{"agora:idea@v1.0.0"},
				CreatedAt: baseTime,
				UpdatedAt: updatedTime,
			},
		},
		{
			name: "Success/AddModule",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "1.0.0"}},
				{resp: &dao.Module{ID: "concept", Namespace: "agora", Version: "1.0.0"}},
			},
			expectModuleSelect: 2,

			projectUpdateMock: &projectUpdateMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: updatedTime,
				},
			},

			schemaInsertMocks: []schemaInsertMock{
				{
					resp: &dao.Schema{
						ID:              uuid.MustParse("00000000-0000-0000-0000-000000000200"),
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "concept",
						ModuleNamespace: "agora",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{},
						CreatedAt:       updatedTime,
					},
				},
			},

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangEN,
				Title:     "Test Project",
				Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
				CreatedAt: baseTime,
				UpdatedAt: updatedTime,
			},
		},
		{
			name: "Success/RemoveModule",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "1.0.0"}},
			},
			expectModuleSelect: 1,

			projectUpdateMock: &projectUpdateMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: updatedTime,
				},
			},

			schemaInsertMocks: []schemaInsertMock{
				{
					resp: &dao.Schema{
						ID:              uuid.MustParse("00000000-0000-0000-0000-000000000200"),
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "concept",
						ModuleNamespace: "agora",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            nil,
						CreatedAt:       updatedTime,
					},
				},
			},

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangEN,
				Title:     "Test Project",
				Workflow:  []string{"agora:idea@v1.0.0"},
				CreatedAt: baseTime,
				UpdatedAt: updatedTime,
			},
		},
		{
			name: "Success/AddAndRemoveModules",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0", "agora:characters@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "1.0.0"}},
				{resp: &dao.Module{ID: "characters", Namespace: "agora", Version: "1.0.0"}},
			},
			expectModuleSelect: 2,

			projectUpdateMock: &projectUpdateMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0", "agora:characters@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: updatedTime,
				},
			},

			schemaInsertMocks: []schemaInsertMock{
				{
					resp: &dao.Schema{
						ID:              uuid.MustParse("00000000-0000-0000-0000-000000000200"),
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "characters",
						ModuleNamespace: "agora",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{},
						CreatedAt:       updatedTime,
					},
				},
				{
					resp: &dao.Schema{
						ID:              uuid.MustParse("00000000-0000-0000-0000-000000000201"),
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "concept",
						ModuleNamespace: "agora",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            nil,
						CreatedAt:       updatedTime,
					},
				},
			},

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangEN,
				Title:     "Test Project",
				Workflow:  []string{"agora:idea@v1.0.0", "agora:characters@v1.0.0"},
				CreatedAt: baseTime,
				UpdatedAt: updatedTime,
			},
		},
		{
			name: "Error/InvalidRequest/EmptyWorkflow",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/EmptyTitle",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModule",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"invalid-module"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectSelect/NotFound",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				err: dao.ErrProjectSelectNotFound,
			},

			expectErr: dao.ErrProjectSelectNotFound,
		},
		{
			name: "Error/ProjectSelect/Generic",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/Forbidden/UserNotOwner",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   otherUserID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expectErr: services.ErrUserDoesNotOwnProject,
		},
		{
			name: "Error/ModuleNotFound",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{err: dao.ErrModuleSelectNotFound},
			},
			expectModuleSelect: 1,

			expectErr: dao.ErrModuleSelectNotFound,
		},
		{
			name: "Error/ForbiddenModuleUpgrade",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v2.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "2.0.0"}},
			},
			expectModuleSelect: 1,

			expectErr: services.ErrForbiddenModuleUpgrade,
		},
		{
			name: "Error/ProjectUpdate",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "1.0.0"}},
			},
			expectModuleSelect: 1,

			projectUpdateMock: &projectUpdateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaInsert",

			request: &services.ProjectUpdateRequest{
				ID:       projectID,
				UserID:   ownerID,
				Title:    "Test Project",
				Workflow: []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMocks: []moduleSelectMock{
				{resp: &dao.Module{ID: "idea", Namespace: "agora", Version: "1.0.0"}},
				{resp: &dao.Module{ID: "concept", Namespace: "agora", Version: "1.0.0"}},
			},
			expectModuleSelect: 2,

			projectUpdateMock: &projectUpdateMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: updatedTime,
				},
			},

			schemaInsertMocks: []schemaInsertMock{
				{
					err: errFoo,
				},
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				projectUpdateRepositorySelect := servicesmocks.NewMockProjectUpdateRepositorySelect(t)
				projectUpdateRepository := servicesmocks.NewMockProjectUpdateRepository(t)
				projectUpdateRepositorySchemaInsert := servicesmocks.NewMockProjectUpdateRepositorySchemaInsert(t)
				moduleSelectRepository := servicesmocks.NewMockProjectUpdateRepositoryModuleSelect(t)

				if testCase.projectSelectMock != nil {
					projectUpdateRepositorySelect.EXPECT().
						Exec(mock.Anything, &dao.ProjectSelectRequest{
							ID: testCase.request.ID,
						}).
						Return(testCase.projectSelectMock.resp, testCase.projectSelectMock.err)
				}

				for i := range testCase.expectModuleSelect {
					moduleSelectRepository.EXPECT().
						Exec(mock.Anything, mock.Anything).
						Return(testCase.moduleSelectMocks[i].resp, testCase.moduleSelectMocks[i].err).
						Once()
				}

				if testCase.projectUpdateMock != nil {
					projectUpdateRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectUpdateRequest) bool {
							return req.ID == testCase.request.ID &&
								req.Title == testCase.request.Title &&
								len(req.Workflow) == len(testCase.request.Workflow)
						})).
						Return(testCase.projectUpdateMock.resp, testCase.projectUpdateMock.err)
				}

				for _, schemaInsertMock := range testCase.schemaInsertMocks {
					projectUpdateRepositorySchemaInsert.EXPECT().
						Exec(mock.Anything, mock.Anything).
						Return(schemaInsertMock.resp, schemaInsertMock.err).
						Once()
				}

				service := services.NewProjectUpdate(
					projectUpdateRepository,
					projectUpdateRepositorySelect,
					projectUpdateRepositorySchemaInsert,
					moduleSelectRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				projectUpdateRepositorySelect.AssertExpectations(t)
				projectUpdateRepository.AssertExpectations(t)
				projectUpdateRepositorySchemaInsert.AssertExpectations(t)
				moduleSelectRepository.AssertExpectations(t)
			})
		})
	}
}
