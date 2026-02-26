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

func TestSchemaRewrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	schemaID := uuid.MustParse("00000000-0000-0000-0000-000000000200")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updateTime := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)

	type schemaRewriteMock struct {
		resp *dao.Schema
		err  error
	}

	type projectSelectMock struct {
		resp *dao.Project
		err  error
	}

	type schemaSelectMock struct {
		resp *dao.Schema
		err  error
	}

	type moduleSelectMock struct {
		resp *dao.Module
		err  error
	}

	testCases := []struct {
		name string

		request *services.SchemaRewriteRequest

		schemaRewriteMock *schemaRewriteMock
		projectSelectMock *projectSelectMock
		schemaSelectMock  *schemaSelectMock
		moduleSelectMock  *moduleSelectMock

		expect    *services.Schema
		expectErr error
	}{
		{
			name: "Success",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
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

			schemaRewriteMock: &schemaRewriteMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Updated Title"},
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
				Data:            map[string]any{"field1": "Updated Title"},
				CreatedAt:       baseTime,
			},
		},
		{
			name: "Success/WithPreversion",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Beta Updated"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:               schemaID,
					ProjectID:        projectID,
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "-beta-1",
					Source:           dao.SchemaSourceAI,
					Data:             map[string]any{"field1": "Beta Original"},
					CreatedAt:        baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
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

			schemaRewriteMock: &schemaRewriteMock{
				resp: &dao.Schema{
					ID:               schemaID,
					ProjectID:        projectID,
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "-beta-1",
					Source:           dao.SchemaSourceAI,
					Data:             map[string]any{"field1": "Beta Updated"},
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
				Data:             map[string]any{"field1": "Beta Updated"},
				CreatedAt:        baseTime,
			},
		},
		{
			name: "Error/SchemaSelect",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SchemaDataIsNil",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            nil, // Null data indicates deleted module entry
					CreatedAt:       baseTime,
				},
			},

			expectErr: dao.ErrSchemaSelectNotFound,
		},
		{
			name: "Error/ProjectSelect",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectOwnership",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: otherUserID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				},
			},

			expectErr: services.ErrUserDoesNotOwnProject,
		},
		{
			name: "Error/ModuleSelect",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
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
			name: "Error/InvalidData",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": 123}, // field1 must be a string
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
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

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/SchemaRewrite",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{"field1": "Updated Title"},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
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

			schemaRewriteMock: &schemaRewriteMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/NilData",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Now:    updateTime,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Success/EmptyData",

			request: &services.SchemaRewriteRequest{
				ID:     schemaID,
				UserID: ownerID,
				Data:   map[string]any{},
				Now:    updateTime,
			},

			schemaSelectMock: &schemaSelectMock{
				resp: &dao.Schema{
					ID:              schemaID,
					ProjectID:       projectID,
					Owner:           &ownerID,
					ModuleID:        "test-module",
					ModuleNamespace: "test-namespace",
					ModuleVersion:   "1.0.0",
					Source:          dao.SchemaSourceUser,
					Data:            map[string]any{"field1": "Original Title"},
					CreatedAt:       baseTime,
				},
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:        projectID,
					Owner:     ownerID,
					Lang:      config.LangEN,
					Title:     "Test Project",
					Workflow:  []string{"test-module"},
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

			schemaRewriteMock: &schemaRewriteMock{
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

				schemaRewriteRepository := servicesmocks.NewMockSchemaRewriteRepository(t)
				projectSelectRepository := servicesmocks.NewMockSchemaRewriteRepositoryProjectSelect(t)
				schemaSelectRepository := servicesmocks.NewMockSchemaRewriteRepositorySchemaSelect(t)
				moduleSelectRepository := servicesmocks.NewMockSchemaRewriteRepositoryModuleSelect(t)

				if testCase.schemaSelectMock != nil {
					schemaSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.SchemaSelectRequest{
							ID: &testCase.request.ID,
						}).
						Return(testCase.schemaSelectMock.resp, testCase.schemaSelectMock.err)
				}

				if testCase.projectSelectMock != nil {
					schemaSelectResp := testCase.schemaSelectMock.resp
					projectSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.ProjectSelectRequest{
							ID: schemaSelectResp.ProjectID,
						}).
						Return(testCase.projectSelectMock.resp, testCase.projectSelectMock.err)
				}

				if testCase.moduleSelectMock != nil {
					schemaSelectResp := testCase.schemaSelectMock.resp
					moduleSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleSelectRequest{
							ID:         schemaSelectResp.ModuleID,
							Namespace:  schemaSelectResp.ModuleNamespace,
							Version:    schemaSelectResp.ModuleVersion,
							Preversion: schemaSelectResp.ModulePreversion,
						}).
						Return(testCase.moduleSelectMock.resp, testCase.moduleSelectMock.err)
				}

				if testCase.schemaRewriteMock != nil {
					schemaRewriteRepository.EXPECT().
						Exec(mock.Anything, &dao.SchemaUpdateRequest{
							ID:   testCase.request.ID,
							Data: testCase.request.Data,
							Now:  testCase.request.Now,
						}).
						Return(testCase.schemaRewriteMock.resp, testCase.schemaRewriteMock.err)
				}

				service := services.NewSchemaRewrite(
					schemaRewriteRepository,
					projectSelectRepository,
					schemaSelectRepository,
					moduleSelectRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				schemaRewriteRepository.AssertExpectations(t)
				projectSelectRepository.AssertExpectations(t)
				schemaSelectRepository.AssertExpectations(t)
				moduleSelectRepository.AssertExpectations(t)
			})
		})
	}
}
