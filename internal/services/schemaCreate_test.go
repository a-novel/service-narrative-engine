package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/samber/lo"
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

func TestSchemaCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	schemaID := uuid.MustParse("00000000-0000-0000-0000-000000000200")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	// Simple JSON schema that requires a "title" string field.
	testModuleSchema := jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"title": {
				Type: "string",
			},
		},
		Required: []string{"title"},
	}

	type schemaInsertMock struct {
		resp *dao.Schema
		err  error
	}

	type projectSelectMock struct {
		resp *dao.Project
		err  error
	}

	type moduleSelectMock struct {
		resp *dao.Module
		err  error
	}

	testCases := []struct {
		name string

		request *services.SchemaCreateRequest

		schemaInsertMock  *schemaInsertMock
		projectSelectMock *projectSelectMock
		moduleSelectMock  *moduleSelectMock

		expect    *services.Schema
		expectErr error
	}{
		{
			name: "Success",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
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

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:        "test-module",
					Namespace: "test-namespace",
					Version:   "1.0.0",
					Schema:    testModuleSchema,
					CreatedAt: baseTime,
				},
			},

			schemaInsertMock: &schemaInsertMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"title": "Test Title"},
					CreatedAt:       baseTime,
				},
			},

			expect: &services.Schema{
				ID:              schemaID,
				ProjectID:       projectID,
				Owner:           &ownerID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				ModuleVersion:   "1.0.0",
				Source:          "USER",
				Data:            map[string]any{"title": "Test Title"},
				CreatedAt:       baseTime,
			},
		},
		{
			name: "Success/WithPreversion",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0-beta-1",
				Source:    "AI",
				Data:      map[string]any{"title": "Beta Test"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-namespace:test-module@v1.0.0-beta-1"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:         "test-module",
					Namespace:  "test-namespace",
					Version:    "1.0.0",
					Preversion: "-beta-1",
					Schema:     testModuleSchema,
					CreatedAt:  baseTime,
				},
			},

			schemaInsertMock: &schemaInsertMock{
				resp: &dao.Schema{
					ID:               schemaID,
					ProjectID:        projectID,
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "-beta-1",
					Source:           dao.SchemaSourceAI,
					Data:             map[string]any{"title": "Beta Test"},
					CreatedAt:        baseTime,
				},
			},

			expect: &services.Schema{
				ID:               schemaID,
				ProjectID:        projectID,
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "1.0.0",
				ModulePreversion: "-beta-1",
				Source:           "AI",
				Data:             map[string]any{"title": "Beta Test"},
				CreatedAt:        baseTime,
			},
		},
		{
			name: "Error/InvalidRequest/MissingModule",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleFormat",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "invalid-module-format",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleVersion",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "namespace:module@invalid",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingSource",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "",
				Data:      map[string]any{"title": "Test Title"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidSource",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "INVALID",
				Data:      map[string]any{"title": "Test Title"},
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectSelect",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
			},

			projectSelectMock: &projectSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectOwnership",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    otherUserID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
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
		{
			name: "Error/ModuleNotInProject",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"other-namespace:other-module@v1.0.0"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expectErr: services.ErrModuleNotInProject,
		},
		{
			name: "Error/ModuleSelect",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
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

			moduleSelectMock: &moduleSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaInsert",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{"title": "Test Title"},
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

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:        "test-module",
					Namespace: "test-namespace",
					Version:   "1.0.0",
					Schema:    testModuleSchema,
					CreatedAt: baseTime,
				},
			},

			schemaInsertMock: &schemaInsertMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/NilData",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Success/EmptyData",

			request: &services.SchemaCreateRequest{
				ID:        schemaID,
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Source:    "USER",
				Data:      map[string]any{},
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

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:        "test-module",
					Namespace: "test-namespace",
					Version:   "1.0.0",
					Schema:    testModuleSchema,
					CreatedAt: baseTime,
				},
			},

			schemaInsertMock: &schemaInsertMock{
				resp: &dao.Schema{
					ID:              schemaID,
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

			expect: &services.Schema{
				ID:              schemaID,
				ProjectID:       projectID,
				Owner:           &ownerID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				ModuleVersion:   "1.0.0",
				Source:          "USER",
				Data:            map[string]any{},
				CreatedAt:       baseTime,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				schemaInsertRepository := servicesmocks.NewMockSchemaCreateRepository(t)
				projectSelectRepository := servicesmocks.NewMockSchemaCreateRepositoryProjectSelect(t)
				moduleSelectRepository := servicesmocks.NewMockSchemaCreateRepositoryModuleSelect(t)

				if testCase.schemaInsertMock != nil {
					schemaInsertRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.SchemaInsertRequest) bool {
							return req.ProjectID == testCase.request.ProjectID &&
								lo.FromPtr(req.Owner) == testCase.request.UserID &&
								req.ModuleID == testCase.moduleSelectMock.resp.ID &&
								req.ModuleNamespace == testCase.moduleSelectMock.resp.Namespace &&
								req.ModuleVersion == testCase.moduleSelectMock.resp.Version &&
								req.ModulePreversion == testCase.moduleSelectMock.resp.Preversion &&
								req.Source == dao.SchemaSource(testCase.request.Source) &&
								assert.Equal(t, req.Data, testCase.request.Data) &&
								time.Since(req.Now) < time.Minute
						})).
						Return(testCase.schemaInsertMock.resp, testCase.schemaInsertMock.err)
				}

				if testCase.projectSelectMock != nil {
					projectSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.ProjectSelectRequest{
							ID: testCase.request.ProjectID,
						}).
						Return(testCase.projectSelectMock.resp, testCase.projectSelectMock.err)
				}

				if testCase.moduleSelectMock != nil {
					decodedModule := lib.DecodeModule(testCase.request.Module)
					moduleSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleSelectRequest{
							ID:         decodedModule.Module,
							Namespace:  decodedModule.Namespace,
							Version:    decodedModule.Version,
							Preversion: decodedModule.Preversion,
						}).
						Return(testCase.moduleSelectMock.resp, testCase.moduleSelectMock.err)
				}

				service := services.NewSchemaCreate(
					schemaInsertRepository,
					projectSelectRepository,
					moduleSelectRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				schemaInsertRepository.AssertExpectations(t)
				projectSelectRepository.AssertExpectations(t)
				moduleSelectRepository.AssertExpectations(t)
			})
		})
	}
}
