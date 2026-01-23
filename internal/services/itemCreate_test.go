package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/services"
	servicesmocks "github.com/a-novel/service-template/internal/services/mocks"
)

func TestItemCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Item
		err  error
	}

	testCases := []struct {
		name string

		request *services.ItemCreateRequest

		repositoryMock *repositoryMock

		expect    *services.Item
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ItemCreateRequest{
				Name:        "test item",
				Description: "test description",
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
			name: "Error/EmptyName",

			request:   &services.ItemCreateRequest{Name: ""},
			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/WhitespaceOnlyName",

			request:   &services.ItemCreateRequest{Name: "   "},
			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/NameTooLong",

			request:   &services.ItemCreateRequest{Name: string(make([]byte, 257))},
			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/Repository",

			request: &services.ItemCreateRequest{Name: "test item"},

			repositoryMock: &repositoryMock{err: errFoo},
			expectErr:      errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := servicesmocks.NewMockItemCreateRepository(t)

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ItemCreateRequest) bool {
						return assert.NotEqual(t, uuid.Nil, req.ID) &&
							assert.WithinDuration(t, time.Now(), req.Now, time.Minute) &&
							assert.Equal(t, testCase.request.Name, req.Name) &&
							assert.Equal(t, testCase.request.Description, req.Description)
					})).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			service := services.NewItemCreate(repository)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
		})
	}
}
