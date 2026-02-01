package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestModuleListNamespaces(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type moduleListNamespacesMock struct {
		resp []string
		err  error
	}

	testCases := []struct {
		name string

		request *services.ModuleListNamespacesRequest

		moduleListNamespacesMock *moduleListNamespacesMock

		expect    []string
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ModuleListNamespacesRequest{
				Limit:  10,
				Offset: 0,
			},

			moduleListNamespacesMock: &moduleListNamespacesMock{
				resp: []string{"alpha", "beta", "gamma"},
			},

			expect: []string{"alpha", "beta", "gamma"},
		},
		{
			name: "Success/WithOffset",

			request: &services.ModuleListNamespacesRequest{
				Limit:  10,
				Offset: 5,
			},

			moduleListNamespacesMock: &moduleListNamespacesMock{
				resp: []string{"beta", "gamma"},
			},

			expect: []string{"beta", "gamma"},
		},
		{
			name: "Success/EmptyResult",

			request: &services.ModuleListNamespacesRequest{
				Limit:  10,
				Offset: 0,
			},

			moduleListNamespacesMock: &moduleListNamespacesMock{
				resp: []string{},
			},

			expect: []string{},
		},
		{
			name: "Error/InvalidRequest/MissingLimit",

			request: &services.ModuleListNamespacesRequest{
				Limit:  0,
				Offset: 0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/LimitTooLarge",

			request: &services.ModuleListNamespacesRequest{
				Limit:  129,
				Offset: 0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetNegative",

			request: &services.ModuleListNamespacesRequest{
				Limit:  10,
				Offset: -1,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/RepositoryError",

			request: &services.ModuleListNamespacesRequest{
				Limit:  10,
				Offset: 0,
			},

			moduleListNamespacesMock: &moduleListNamespacesMock{
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

				moduleListNamespacesRepository := servicesmocks.NewMockModuleListNamespacesRepository(t)

				if testCase.moduleListNamespacesMock != nil {
					moduleListNamespacesRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleListNamespacesRequest{
							Limit:  testCase.request.Limit,
							Offset: testCase.request.Offset,
						}).
						Return(testCase.moduleListNamespacesMock.resp, testCase.moduleListNamespacesMock.err)
				}

				service := services.NewModuleListNamespaces(
					moduleListNamespacesRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				moduleListNamespacesRepository.AssertExpectations(t)
			})
		})
	}
}
