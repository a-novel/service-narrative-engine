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

func TestModuleListIDs(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type moduleListIDsMock struct {
		resp []string
		err  error
	}

	testCases := []struct {
		name string

		request *services.ModuleListIDsRequest

		moduleListIDsMock *moduleListIDsMock

		expect    []string
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     10,
				Offset:    0,
			},

			moduleListIDsMock: &moduleListIDsMock{
				resp: []string{"concept", "draft", "idea"},
			},

			expect: []string{"concept", "draft", "idea"},
		},
		{
			name: "Success/WithOffset",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     10,
				Offset:    5,
			},

			moduleListIDsMock: &moduleListIDsMock{
				resp: []string{"draft", "idea"},
			},

			expect: []string{"draft", "idea"},
		},
		{
			name: "Success/EmptyResult",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     10,
				Offset:    0,
			},

			moduleListIDsMock: &moduleListIDsMock{
				resp: []string{},
			},

			expect: []string{},
		},
		{
			name: "Error/InvalidRequest/MissingNamespace",

			request: &services.ModuleListIDsRequest{
				Limit:  10,
				Offset: 0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingLimit",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     0,
				Offset:    0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/LimitTooLarge",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     129,
				Offset:    0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetNegative",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     10,
				Offset:    -1,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/RepositoryError",

			request: &services.ModuleListIDsRequest{
				Namespace: "agora",
				Limit:     10,
				Offset:    0,
			},

			moduleListIDsMock: &moduleListIDsMock{
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

				moduleListIDsRepository := servicesmocks.NewMockModuleListIDsRepository(t)

				if testCase.moduleListIDsMock != nil {
					moduleListIDsRepository.EXPECT().
						Exec(mock.Anything, &dao.ModuleListIDsRequest{
							Namespace: testCase.request.Namespace,
							Limit:     testCase.request.Limit,
							Offset:    testCase.request.Offset,
						}).
						Return(testCase.moduleListIDsMock.resp, testCase.moduleListIDsMock.err)
				}

				service := services.NewModuleListIDs(
					moduleListIDsRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				moduleListIDsRepository.AssertExpectations(t)
			})
		})
	}
}
