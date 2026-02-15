package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestProjectInit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	schema1ID := uuid.MustParse("00000000-0000-0000-0000-000000000200")
	schema2ID := uuid.MustParse("00000000-0000-0000-0000-000000000201")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	type projectInsertMock struct {
		resp *dao.Project
		err  error
	}

	type schemaInsertMock struct {
		resp *dao.Schema
		err  error
	}

	type moduleExistsMock struct {
		resp bool
		err  error
	}

	testCases := []struct {
		name string

		request *services.ProjectInitRequest

		moduleExistsMocks  []*moduleExistsMock
		expectModuleExists int
		projectInsertMock  *projectInsertMock
		schemaInsertMocks  []*schemaInsertMock
		expectSchemaInsert int

		expect    *services.Project
		expectErr error
	}{
		{
			name: "Success/SingleModule",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			moduleExistsMocks: []*moduleExistsMock{
				{resp: true},
			},
			expectModuleExists: 1,

			projectInsertMock: &projectInsertMock{
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

			schemaInsertMocks: []*schemaInsertMock{
				{
					resp: &dao.Schema{
						ID:              schema1ID,
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "test-module",
						ModuleNamespace: "test-namespace",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{},
						CreatedAt:       baseTime,
					},
				},
			},

			expectSchemaInsert: 1,

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
			name: "Success/MultipleModules",

			request: &services.ProjectInitRequest{
				Owner: ownerID,
				Lang:  config.LangFR,
				Title: "Mon Projet",
				Workflow: []string{
					"agora:idea@v1.0.0",
					"agora:concept@v2.1.3",
				},
			},

			moduleExistsMocks: []*moduleExistsMock{
				{resp: true},
				{resp: true},
			},
			expectModuleExists: 2,

			projectInsertMock: &projectInsertMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangFR,
					Title:     "Mon Projet",
					Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v2.1.3"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			schemaInsertMocks: []*schemaInsertMock{
				{
					resp: &dao.Schema{
						ID:              schema1ID,
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "idea",
						ModuleNamespace: "agora",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{},
						CreatedAt:       baseTime,
					},
				},
				{
					resp: &dao.Schema{
						ID:              schema2ID,
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "concept",
						ModuleNamespace: "agora",
						ModuleVersion:   "2.1.3",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{},
						CreatedAt:       baseTime,
					},
				},
			},

			expectSchemaInsert: 2,

			expect: &services.Project{
				ID:        projectID,
				Owner:     ownerID,
				Lang:      config.LangFR,
				Title:     "Mon Projet",
				Workflow:  []string{"agora:idea@v1.0.0", "agora:concept@v2.1.3"},
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			},
		},
		{
			name: "Error/InvalidRequest/MissingLang",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     "",
				Title:    "Test Project",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidLang",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     "invalid",
				Title:    "Test Project",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingTitle",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/EmptyModules",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleFormat",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{"invalid-module-format"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleVersion",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{"namespace:module@invalid"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ModuleNotFound",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			moduleExistsMocks: []*moduleExistsMock{
				{resp: false},
			},
			expectModuleExists: 1,

			expectErr: dao.ErrModuleSelectNotFound,
		},
		{
			name: "Error/ProjectInsert",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			moduleExistsMocks: []*moduleExistsMock{
				{resp: true},
			},
			expectModuleExists: 1,

			projectInsertMock: &projectInsertMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaInsert/FirstModule",

			request: &services.ProjectInitRequest{
				Owner:    ownerID,
				Lang:     config.LangEN,
				Title:    "Test Project",
				Workflow: []string{"test-namespace:test-module@v1.0.0"},
			},

			moduleExistsMocks: []*moduleExistsMock{
				{resp: true},
			},
			expectModuleExists: 1,

			projectInsertMock: &projectInsertMock{
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

			schemaInsertMocks: []*schemaInsertMock{
				{
					err: errFoo,
				},
			},

			expectSchemaInsert: 1,

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaInsert/SecondModule",

			request: &services.ProjectInitRequest{
				Owner: ownerID,
				Lang:  config.LangEN,
				Title: "Test Project",
				Workflow: []string{
					"agora:idea@v1.0.0",
					"agora:concept@v2.1.3",
				},
			},

			moduleExistsMocks: []*moduleExistsMock{
				{resp: true},
				{resp: true},
			},
			expectModuleExists: 2,

			projectInsertMock: &projectInsertMock{
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

			schemaInsertMocks: []*schemaInsertMock{
				{
					resp: &dao.Schema{
						ID:              schema1ID,
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "idea",
						ModuleNamespace: "agora",
						ModuleVersion:   "1.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{},
						CreatedAt:       baseTime,
					},
				},
				{
					err: errFoo,
				},
			},

			expectSchemaInsert: 2,

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				projectInsertRepository := servicesmocks.NewMockProjectInsertRepository(t)
				schemaInsertRepository := servicesmocks.NewMockProjectInsertRepositorySchemaInsert(t)
				moduleExistsRepository := servicesmocks.NewMockProjectInsertRepositoryModuleExists(t)

				for i := range testCase.expectModuleExists {
					func(idx int) {
						mockData := testCase.moduleExistsMocks[idx]
						decodedModule := lib.DecodeModule(testCase.request.Workflow[idx])

						moduleExistsRepository.EXPECT().
							Exec(mock.Anything, mock.MatchedBy(func(req *dao.ModuleSelectRequest) bool {
								return req.ID == decodedModule.Module &&
									req.Namespace == decodedModule.Namespace &&
									req.Version == decodedModule.Version &&
									req.Preversion == decodedModule.Preversion
							})).
							Return(mockData.resp, mockData.err).
							Once()
					}(i)
				}

				if testCase.projectInsertMock != nil {
					projectInsertRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectInsertRequest) bool {
							return assert.NotEqual(t, uuid.Nil, req.ID) &&
								assert.Equal(t, testCase.request.Owner, req.Owner) &&
								assert.Equal(t, testCase.request.Lang, req.Lang) &&
								assert.Equal(t, testCase.request.Title, req.Title) &&
								assert.Equal(t, testCase.request.Workflow, req.Workflow) &&
								assert.WithinDuration(t, time.Now(), req.Now, time.Minute)
						})).
						Return(testCase.projectInsertMock.resp, testCase.projectInsertMock.err)
				}

				for i := range testCase.expectSchemaInsert {
					func(idx int) {
						mockData := testCase.schemaInsertMocks[idx]
						decodedModule := lib.DecodeModule(testCase.request.Workflow[idx])

						schemaInsertRepository.EXPECT().
							Exec(mock.Anything, mock.MatchedBy(func(req *dao.SchemaInsertRequest) bool {
								return req.ID != uuid.Nil &&
									req.ProjectID == testCase.projectInsertMock.resp.ID &&
									*req.Owner == testCase.request.Owner &&
									req.ModuleID == decodedModule.Module &&
									req.ModuleNamespace == decodedModule.Namespace &&
									req.ModuleVersion == decodedModule.Version &&
									req.Source == dao.SchemaSourceUser &&
									len(req.Data) == 0 &&
									time.Since(req.Now) < time.Minute
							})).
							Return(mockData.resp, mockData.err).
							Once()
					}(i)
				}

				service := services.NewProjectInit(projectInsertRepository, schemaInsertRepository, moduleExistsRepository)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				projectInsertRepository.AssertExpectations(t)
				schemaInsertRepository.AssertExpectations(t)
				moduleExistsRepository.AssertExpectations(t)
			})
		})
	}
}
