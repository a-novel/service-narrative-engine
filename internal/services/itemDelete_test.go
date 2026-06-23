package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestItemDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Item
		err  error
	}

	testCases := []struct {
		name string

		request *services.ItemDeleteRequest

		repositoryMock *repositoryMock

		expect    *services.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ItemDeleteRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Item{
					ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Name:        "test item",
					Description: "test description",
					CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Item{
				ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:        "test item",
				Description: "test description",
				CreatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/InvalidID",

			request:   &services.ItemDeleteRequest{ID: uuid.Nil},
			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/Repository",

			request: &services.ItemDeleteRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			repositoryMock: &repositoryMock{err: errFoo},
			expectErr:      errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := servicesmocks.NewMockItemDeleteRepository(t)

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, &dao.ItemDeleteRequest{
						ID: testCase.request.ID,
					}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			service := services.NewItemDelete(repository)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
		})
	}
}
