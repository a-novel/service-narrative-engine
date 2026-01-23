package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models"
	"github.com/a-novel/service-narrative-engine/internal/models/modules"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestModuleLoadSystem(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	testModuleSchema := jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"title": {
				Type: "string",
			},
			"count": {
				Type: "integer",
			},
		},
		Required: []string{"title"},
	}

	testModuleSchemaUpdated := jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"title": {
				Type: "string",
			},
			"description": {
				Type: "string",
			},
		},
		Required: []string{"title", "description"},
	}

	testModuleUI := models.ModuleUi{
		Component: "test-component",
		Params:    map[string]any{"key": "value"},
		Target:    "title",
	}

	testModuleUIUpdated := models.ModuleUi{
		Component: "updated-component",
		Params:    map[string]any{"newKey": "newValue"},
		Target:    "description",
	}

	validDescription := "This is a valid description that is at least 32 characters long."

	type moduleInsertMock struct {
		resp *dao.Module
		err  error
	}

	type moduleListVersionsMock struct {
		resp []*dao.ModuleVersion
		err  error
	}

	type moduleSelectMock struct {
		resp *dao.Module
		err  error
	}

	testCases := []struct {
		name string

		request *services.ModuleLoadSystemRequest

		moduleInsertMock       *moduleInsertMock
		moduleListVersionsMock *moduleListVersionsMock
		moduleSelectMock       *moduleSelectMock

		expect    *services.Module
		expectErr error
	}{
		{
			name: "Success/NonDevMode",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: false,
			},

			moduleInsertMock: &moduleInsertMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
					CreatedAt:   baseTime,
				},
			},

			expect: &services.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				CreatedAt:   baseTime,
			},
		},
		{
			name: "Success/DevMode/NoExistingVersions",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{},
			},

			moduleInsertMock: &moduleInsertMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "uuid-preversion",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
					CreatedAt:   baseTime,
				},
			},

			expect: &services.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "uuid-preversion",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				CreatedAt:   baseTime,
			},
		},
		{
			name: "Success/DevMode/ExistingVersionWithDifferentSchema",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchemaUpdated,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "old-preversion",
						CreatedAt:  baseTime,
					},
				},
			},

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "old-preversion",
					Description: validDescription,
					Schema:      testModuleSchema, // Different from testModuleSchemaUpdated.
					UI:          testModuleUI,
					CreatedAt:   baseTime,
				},
			},

			moduleInsertMock: &moduleInsertMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "new-uuid-preversion",
					Description: validDescription,
					Schema:      testModuleSchemaUpdated,
					UI:          testModuleUI,
					CreatedAt:   baseTime,
				},
			},

			expect: &services.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "new-uuid-preversion",
				Description: validDescription,
				Schema:      testModuleSchemaUpdated,
				UI:          testModuleUI,
				CreatedAt:   baseTime,
			},
		},
		{
			name: "Success/DevMode/ExistingVersionWithDifferentUI",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUIUpdated,
				},
				Version: "1.0.0",
				DevMode: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "old-preversion",
						CreatedAt:  baseTime,
					},
				},
			},

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "old-preversion",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI, // Different from testModuleUIUpdated.
					CreatedAt:   baseTime,
				},
			},

			moduleInsertMock: &moduleInsertMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "new-uuid-preversion",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUIUpdated,
					CreatedAt:   baseTime,
				},
			},

			expect: &services.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "new-uuid-preversion",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUIUpdated,
				CreatedAt:   baseTime,
			},
		},
		{
			name: "Success/DevMode/ExistingVersionNoChanges",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "existing-preversion",
						CreatedAt:  baseTime,
					},
				},
			},

			moduleSelectMock: &moduleSelectMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "existing-preversion",
					Description: validDescription,
					Schema:      testModuleSchema, // Must be resolved to match service comparison.
					UI:          testModuleUI,
					CreatedAt:   baseTime,
				},
			},

			// No insert mock - existing module should be returned.

			expect: &services.Module{
				ID:          "test-module",
				Namespace:   "test-namespace",
				Version:     "1.0.0",
				Preversion:  "existing-preversion",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				CreatedAt:   baseTime,
			},
		},
		{
			name: "Error/InvalidRequest/MissingVersion",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "",
				DevMode: false,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidVersionFormat",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "invalid-version",
				DevMode: false,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/DevMode/ListVersionsFailure",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/DevMode/SelectFailure",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "existing-preversion",
						CreatedAt:  baseTime,
					},
				},
			},

			moduleSelectMock: &moduleSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertFailure",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: false,
			},

			moduleInsertMock: &moduleInsertMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/InsertAlreadyExists",

			request: &services.ModuleLoadSystemRequest{
				Module: modules.SystemModule{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Description: validDescription,
					Schema:      testModuleSchema,
					UI:          testModuleUI,
				},
				Version: "1.0.0",
				DevMode: false,
			},

			moduleInsertMock: &moduleInsertMock{
				err: dao.ErrModuleInsertAlreadyExists,
			},

			expectErr: dao.ErrModuleInsertAlreadyExists,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				moduleInsertRepository := servicesmocks.NewMockModuleLoadSystemRepository(t)
				moduleDeleteRepository := servicesmocks.NewMockModuleLoadSystemRepositoryDelete(t)
				moduleSelectRepository := servicesmocks.NewMockModuleLoadSystemRepositorySelect(t)
				moduleListVersionsRepository := servicesmocks.NewMockModuleLoadSystemRepositoryListVersions(t)

				if testCase.moduleListVersionsMock != nil {
					moduleListVersionsRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleListVersionsRequest{
							ID:         testCase.request.Module.ID,
							Namespace:  testCase.request.Module.Namespace,
							Limit:      1,
							Version:    testCase.request.Version,
							Preversion: true,
						}).
						Return(testCase.moduleListVersionsMock.resp, testCase.moduleListVersionsMock.err)
				}

				if testCase.moduleSelectMock != nil {
					moduleSelectRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.ModuleSelectRequest) bool {
							return req.ID == testCase.request.Module.ID &&
								req.Namespace == testCase.request.Module.Namespace &&
								req.Version == testCase.request.Version
						})).
						Return(testCase.moduleSelectMock.resp, testCase.moduleSelectMock.err)
				}

				if testCase.moduleInsertMock != nil {
					moduleInsertRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.ModuleInsertRequest) bool {
							versionMatch := req.Version == testCase.request.Version
							// In dev mode, preversion should be a non-empty UUID-like string.
							// In non-dev mode, preversion should be empty.
							preversionMatch := (testCase.request.DevMode && req.Preversion != "") ||
								(!testCase.request.DevMode && req.Preversion == "")
							idMatch := req.ID == testCase.request.Module.ID
							namespaceMatch := req.Namespace == testCase.request.Module.Namespace
							descriptionMatch := req.Description == testCase.request.Module.Description
							timeMatch := time.Since(req.Now) < time.Minute

							return versionMatch && preversionMatch && idMatch &&
								namespaceMatch && descriptionMatch && timeMatch
						})).
						Return(testCase.moduleInsertMock.resp, testCase.moduleInsertMock.err)
				}

				service := services.NewModuleLoadSystem(
					moduleInsertRepository,
					moduleDeleteRepository,
					moduleSelectRepository,
					moduleListVersionsRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				require.Equal(t, testCase.expect, resp)

				moduleInsertRepository.AssertExpectations(t)
				moduleDeleteRepository.AssertExpectations(t)
				moduleSelectRepository.AssertExpectations(t)
				moduleListVersionsRepository.AssertExpectations(t)
			})
		})
	}
}
