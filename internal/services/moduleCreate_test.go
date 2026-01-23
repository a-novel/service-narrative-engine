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
	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestModuleCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	testModuleSchema := jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"title": {
				Type: "string",
			},
		},
		Required: []string{"title"},
	}

	testModuleUI := models.ModuleUi{
		Component: "test-component",
		Params:    map[string]any{"key": "value"},
		Target:    "title",
	}

	validDescription := "This is a valid description that is at least 32 characters long."

	type moduleInsertMock struct {
		resp *dao.Module
		err  error
	}

	type moduleDeleteMock struct {
		resp *dao.Module
		err  error
	}

	testCases := []struct {
		name string

		request *services.ModuleCreateRequest

		moduleInsertMock *moduleInsertMock
		moduleDeleteMock *moduleDeleteMock

		expect    *services.Module
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
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
			name: "Success/WithPreversion",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0-beta-1",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			moduleInsertMock: &moduleInsertMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Preversion:  "-beta-1",
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
				Preversion:  "-beta-1",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				CreatedAt:   baseTime,
			},
		},
		{
			name: "Success/OverwriteExisting",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				Overwrite:   true,
			},

			moduleDeleteMock: &moduleDeleteMock{
				resp: &dao.Module{
					ID:          "test-module",
					Namespace:   "test-namespace",
					Version:     "1.0.0",
					Description: "old description that was previously here",
					Schema:      testModuleSchema,
					UI:          testModuleUI,
					CreatedAt:   baseTime,
				},
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
			name: "Success/OverwriteNonExisting",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				Overwrite:   true,
			},

			moduleDeleteMock: &moduleDeleteMock{
				err: dao.ErrModuleDeleteNotFound,
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
			name: "Error/InvalidRequest/MissingModule",

			request: &services.ModuleCreateRequest{
				Module:      "",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleFormat",

			request: &services.ModuleCreateRequest{
				Module:      "invalid-module-format",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleVersion",

			request: &services.ModuleCreateRequest{
				Module:      "namespace:module@invalid",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingDescription",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: "",
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/DescriptionTooShort",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: "Too short",
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/DescriptionTooLong",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: string(make([]byte, 513)),
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidSchema",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema: jsonschema.Schema{
					Ref: "invalid-ref",
				},
				UI: testModuleUI,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ModuleInsert",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			moduleInsertMock: &moduleInsertMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ModuleInsertAlreadyExists",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
			},

			moduleInsertMock: &moduleInsertMock{
				err: dao.ErrModuleInsertAlreadyExists,
			},

			expectErr: dao.ErrModuleInsertAlreadyExists,
		},
		{
			name: "Error/ModuleDeleteFails",

			request: &services.ModuleCreateRequest{
				Module:      "test-namespace:test-module@v1.0.0",
				Description: validDescription,
				Schema:      testModuleSchema,
				UI:          testModuleUI,
				Overwrite:   true,
			},

			moduleDeleteMock: &moduleDeleteMock{
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

				moduleInsertRepository := servicesmocks.NewMockModuleCreateRepository(t)
				moduleDeleteRepository := servicesmocks.NewMockModuleCreateRepositoryDelete(t)

				if testCase.moduleDeleteMock != nil {
					decodedModule := lib.DecodeModule(testCase.request.Module)
					moduleDeleteRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleDeleteRequest{
							ID:         decodedModule.Module,
							Namespace:  decodedModule.Namespace,
							Version:    decodedModule.Version,
							Preversion: decodedModule.Preversion,
						}).
						Return(testCase.moduleDeleteMock.resp, testCase.moduleDeleteMock.err)
				}

				if testCase.moduleInsertMock != nil {
					decodedModule := lib.DecodeModule(testCase.request.Module)
					moduleInsertRepository.EXPECT().
						Exec(mock.Anything, mock.MatchedBy(func(req *dao.ModuleInsertRequest) bool {
							return req.ID == decodedModule.Module &&
								req.Namespace == decodedModule.Namespace &&
								req.Version == decodedModule.Version &&
								req.Preversion == decodedModule.Preversion &&
								req.Description == testCase.request.Description &&
								time.Since(req.Now) < time.Minute
						})).
						Return(testCase.moduleInsertMock.resp, testCase.moduleInsertMock.err)
				}

				service := services.NewModuleCreate(
					moduleInsertRepository,
					moduleDeleteRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				moduleInsertRepository.AssertExpectations(t)
				moduleDeleteRepository.AssertExpectations(t)
			})
		})
	}
}
