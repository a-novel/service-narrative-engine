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

func TestSchemaGenerate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	schemaID := uuid.MustParse("00000000-0000-0000-0000-000000000200")
	otherSchemaID := uuid.MustParse("00000000-0000-0000-0000-000000000201")

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

	type schemaGenerateMock struct {
		resp map[string]any
		err  error
	}

	type schemaListMock struct {
		resp []*dao.Schema
		err  error
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

		request *services.SchemaGenerateRequest

		schemaGenerateMock *schemaGenerateMock
		schemaListMock     *schemaListMock
		schemaInsertMock   *schemaInsertMock
		projectSelectMock  *projectSelectMock
		moduleSelectMock   *moduleSelectMock

		expect    *services.Schema
		expectErr error
	}{
		{
			name: "Success",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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

			schemaListMock: &schemaListMock{
				resp: []*dao.Schema{},
			},

			schemaGenerateMock: &schemaGenerateMock{
				resp: map[string]any{"title": "Generated Title"},
			},

			schemaInsertMock: &schemaInsertMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceAI,
					Data:            map[string]any{"title": "Generated Title"},
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
				Source:          "AI",
				Data:            map[string]any{"title": "Generated Title"},
				CreatedAt:       baseTime,
			},
		},
		{
			name: "Success/WithPreversion",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0-beta-1",
				Lang:      config.LangFR,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangFR,
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

			schemaListMock: &schemaListMock{
				resp: []*dao.Schema{},
			},

			schemaGenerateMock: &schemaGenerateMock{
				resp: map[string]any{"title": "Beta Generated"},
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
					Data:             map[string]any{"title": "Beta Generated"},
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
				Data:             map[string]any{"title": "Beta Generated"},
				CreatedAt:        baseTime,
			},
		},
		{
			name: "Success/WithContext",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-namespace:test-module@v1.0.0", "other-namespace:other-module@v1.0.0"},
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

			schemaListMock: &schemaListMock{
				resp: []*dao.Schema{
					{
						ID:              otherSchemaID,
						ProjectID:       projectID,
						Owner:           &ownerID,
						ModuleID:        "other-module",
						ModuleNamespace: "other-namespace",
						ModuleVersion:   "2.0.0",
						Source:          dao.SchemaSourceUser,
						Data:            map[string]any{"content": "Other content"},
						CreatedAt:       baseTime,
					},
				},
			},

			schemaGenerateMock: &schemaGenerateMock{
				resp: map[string]any{"title": "Generated with Context"},
			},

			schemaInsertMock: &schemaInsertMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceAI,
					Data:            map[string]any{"title": "Generated with Context"},
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
				Source:          "AI",
				Data:            map[string]any{"title": "Generated with Context"},
				CreatedAt:       baseTime,
			},
		},
		{
			name: "Error/InvalidRequest/MissingProjectID",

			request: &services.SchemaGenerateRequest{
				UserID: ownerID,
				Module: "test-namespace:test-module@v1.0.0",
				Lang:   config.LangEN,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingUserID",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingModule",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "",
				Lang:      config.LangEN,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleFormat",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "invalid-module-format",
				Lang:      config.LangEN,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingLang",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      "",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidLang",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      "invalid",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectSelect",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
			},

			projectSelectMock: &projectSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectOwnership",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    otherUserID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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
			name: "Error/SchemaList",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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

			schemaListMock: &schemaListMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaGenerate",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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

			schemaListMock: &schemaListMock{
				resp: []*dao.Schema{},
			},

			schemaGenerateMock: &schemaGenerateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaInsert",

			request: &services.SchemaGenerateRequest{
				ProjectID: projectID,
				UserID:    ownerID,
				Module:    "test-namespace:test-module@v1.0.0",
				Lang:      config.LangEN,
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

			schemaListMock: &schemaListMock{
				resp: []*dao.Schema{},
			},

			schemaGenerateMock: &schemaGenerateMock{
				resp: map[string]any{"title": "Generated Title"},
			},

			schemaInsertMock: &schemaInsertMock{
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

				schemaGenerateRepository := servicesmocks.NewMockSchemaGenerateRepository(t)
				schemaListRepository := servicesmocks.NewMockSchemaGenerateRepositorySchemaList(t)
				schemaInsertRepository := servicesmocks.NewMockSchemaGenerateRepositorySchemaInsert(t)
				projectSelectRepository := servicesmocks.NewMockSchemaGenerateRepositoryProjectSelect(t)
				moduleSelectRepository := servicesmocks.NewMockSchemaGenerateRepositoryModuleSelect(t)

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

				if testCase.schemaListMock != nil {
					schemaListRepository.EXPECT().
						Exec(mock.Anything, &dao.SchemaListRequest{
							ProjectID: testCase.request.ProjectID,
						}).
						Return(testCase.schemaListMock.resp, testCase.schemaListMock.err)
				}

				if testCase.schemaGenerateMock != nil {
					decodedModule := lib.DecodeModule(testCase.request.Module)
					schemaGenerateRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.ModuleGenerateRequest) bool {
							return req.Module.ID == testCase.moduleSelectMock.resp.ID &&
								req.Module.Namespace == testCase.moduleSelectMock.resp.Namespace &&
								req.Module.Version == testCase.moduleSelectMock.resp.Version &&
								req.Module.Preversion == testCase.moduleSelectMock.resp.Preversion &&
								req.Lang == testCase.request.Lang &&
								// Verify that the context excludes the module being generated.
								!lo.ContainsBy(req.Context.([]*dao.Schema), func(s *dao.Schema) bool {
									return s.ModuleNamespace == decodedModule.Namespace && s.ModuleID == decodedModule.Module
								})
						})).
						Return(testCase.schemaGenerateMock.resp, testCase.schemaGenerateMock.err)
				}

				if testCase.schemaInsertMock != nil {
					schemaInsertRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.SchemaInsertRequest) bool {
							return req.ProjectID == testCase.request.ProjectID &&
								lo.FromPtr(req.Owner) == testCase.request.UserID &&
								req.ModuleID == testCase.moduleSelectMock.resp.ID &&
								req.ModuleNamespace == testCase.moduleSelectMock.resp.Namespace &&
								req.ModuleVersion == testCase.moduleSelectMock.resp.Version &&
								req.ModulePreversion == testCase.moduleSelectMock.resp.Preversion &&
								req.Source == dao.SchemaSourceAI &&
								assert.Equal(t, req.Data, testCase.schemaGenerateMock.resp) &&
								time.Since(req.Now) < time.Minute
						})).
						Return(testCase.schemaInsertMock.resp, testCase.schemaInsertMock.err)
				}

				service := services.NewSchemaGenerate(
					schemaGenerateRepository,
					schemaListRepository,
					schemaInsertRepository,
					projectSelectRepository,
					moduleSelectRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				schemaGenerateRepository.AssertExpectations(t)
				schemaListRepository.AssertExpectations(t)
				schemaInsertRepository.AssertExpectations(t)
				projectSelectRepository.AssertExpectations(t)
				moduleSelectRepository.AssertExpectations(t)
			})
		})
	}
}
