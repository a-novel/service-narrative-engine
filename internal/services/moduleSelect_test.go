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

func TestModuleSelect(t *testing.T) {
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

	type moduleSelectMock struct {
		resp *dao.Module
		err  error
	}

	testCases := []struct {
		name string

		request *services.ModuleSelectRequest

		moduleSelectMock *moduleSelectMock

		expect    *services.Module
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ModuleSelectRequest{
				Module: "test-namespace:test-module@v1.0.0",
			},

			moduleSelectMock: &moduleSelectMock{
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

			request: &services.ModuleSelectRequest{
				Module: "test-namespace:test-module@v1.0.0-beta-1",
			},

			moduleSelectMock: &moduleSelectMock{
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
			name: "Error/InvalidRequest/MissingModule",

			request: &services.ModuleSelectRequest{
				Module: "",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleFormat",

			request: &services.ModuleSelectRequest{
				Module: "invalid-module-format",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidModuleVersion",

			request: &services.ModuleSelectRequest{
				Module: "namespace:module@invalid",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/ModuleTooLong",

			request: &services.ModuleSelectRequest{
				Module: "test-namespace:test-module@v1.0.0" + string(make([]byte, 500)),
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ModuleNotFound",

			request: &services.ModuleSelectRequest{
				Module: "test-namespace:test-module@v1.0.0",
			},

			moduleSelectMock: &moduleSelectMock{
				err: dao.ErrModuleSelectNotFound,
			},

			expectErr: dao.ErrModuleSelectNotFound,
		},
		{
			name: "Error/RepositoryError",

			request: &services.ModuleSelectRequest{
				Module: "test-namespace:test-module@v1.0.0",
			},

			moduleSelectMock: &moduleSelectMock{
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

				moduleSelectRepository := servicesmocks.NewMockModuleSelectRepository(t)

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

				service := services.NewModuleSelect(
					moduleSelectRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				moduleSelectRepository.AssertExpectations(t)
			})
		})
	}
}
