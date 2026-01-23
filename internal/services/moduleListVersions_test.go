package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestModuleListVersions(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	type moduleListVersionsMock struct {
		resp []*dao.ModuleVersion
		err  error
	}

	testCases := []struct {
		name string

		request *services.ModuleListVersionsRequest

		moduleListVersionsMock *moduleListVersionsMock

		expect    []*services.ModuleVersion
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "",
						CreatedAt:  baseTime,
					},
				},
			},

			expect: []*services.ModuleVersion{
				{
					Version:    "1.0.0",
					Preversion: "",
					CreatedAt:  baseTime,
				},
			},
		},
		{
			name: "Success/WithVersion",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "",
						CreatedAt:  baseTime,
					},
				},
			},

			expect: []*services.ModuleVersion{
				{
					Version:    "1.0.0",
					Preversion: "",
					CreatedAt:  baseTime,
				},
			},
		},
		{
			name: "Success/MultipleVersions",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "",
						CreatedAt:  baseTime,
					},
					{
						Version:    "1.0.1",
						Preversion: "",
						CreatedAt:  baseTime.Add(time.Hour),
					},
				},
			},

			expect: []*services.ModuleVersion{
				{
					Version:    "1.0.0",
					Preversion: "",
					CreatedAt:  baseTime,
				},
				{
					Version:    "1.0.1",
					Preversion: "",
					CreatedAt:  baseTime.Add(time.Hour),
				},
			},
		},
		{
			name: "Success/WithPreversion",

			request: &services.ModuleListVersionsRequest{
				ID:         "test-module",
				Namespace:  "test-namespace",
				Limit:      10,
				Offset:     0,
				Version:    "1.0.0",
				Preversion: true,
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "-beta-1",
						CreatedAt:  baseTime,
					},
				},
			},

			expect: []*services.ModuleVersion{
				{
					Version:    "1.0.0",
					Preversion: "-beta-1",
					CreatedAt:  baseTime,
				},
			},
		},
		{
			name: "Success/WithOffset",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    5,
				Version:   "1.0.0",
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{
					{
						Version:    "1.0.0",
						Preversion: "",
						CreatedAt:  baseTime,
					},
				},
			},

			expect: []*services.ModuleVersion{
				{
					Version:    "1.0.0",
					Preversion: "",
					CreatedAt:  baseTime,
				},
			},
		},
		{
			name: "Success/EmptyResult",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			moduleListVersionsMock: &moduleListVersionsMock{
				resp: []*dao.ModuleVersion{},
			},

			expect: []*services.ModuleVersion{},
		},
		{
			name: "Error/InvalidRequest/MissingID",

			request: &services.ModuleListVersionsRequest{
				ID:        "",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingNamespace",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/InvalidVersion",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "invalid-version",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/IDTooLong",

			request: &services.ModuleListVersionsRequest{
				ID:        string(make([]byte, 129)),
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/NamespaceTooLong",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: string(make([]byte, 129)),
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingLimit",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     0,
				Offset:    0,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/LimitTooLarge",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     129,
				Offset:    0,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetNegative",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    -1,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetTooLarge",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    8193,
				Version:   "1.0.0",
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/RepositoryError",

			request: &services.ModuleListVersionsRequest{
				ID:        "test-module",
				Namespace: "test-namespace",
				Limit:     10,
				Offset:    0,
				Version:   "1.0.0",
			},

			moduleListVersionsMock: &moduleListVersionsMock{
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

				moduleListVersionsRepository := servicesmocks.NewMockModuleListVersionsRepository(t)

				if testCase.moduleListVersionsMock != nil {
					moduleListVersionsRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleListVersionsRequest{
							ID:         testCase.request.ID,
							Namespace:  testCase.request.Namespace,
							Limit:      testCase.request.Limit,
							Offset:     testCase.request.Offset,
							Version:    testCase.request.Version,
							Preversion: testCase.request.Preversion,
						}).
						Return(testCase.moduleListVersionsMock.resp, testCase.moduleListVersionsMock.err)
				}

				service := services.NewModuleListVersions(
					moduleListVersionsRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				moduleListVersionsRepository.AssertExpectations(t)
			})
		})
	}
}
