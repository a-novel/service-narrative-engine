package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.ProjectListRequest

		daoRequest *dao.ProjectListRequest
		daoMock    *daoMock[[]*dao.Project]

		expect    []*core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectListRequest{OwnerID: ownerID, Limit: 10, After: projectID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 10, After: projectID},
			daoMock:    &daoMock[[]*dao.Project]{resp: []*dao.Project{daoProject()}},

			expect: []*core.Project{
				{
					ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
					Revision: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
				},
			},
		},
		{
			name: "Success/Defaults",

			request: &core.ProjectListRequest{OwnerID: ownerID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: core.ProjectListDefaultSize, After: uuid.Max},
			daoMock:    &daoMock[[]*dao.Project]{resp: []*dao.Project{}},

			expect: []*core.Project{},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.ProjectListRequest{OwnerID: ownerID, Limit: 101},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Internal",

			request: &core.ProjectListRequest{OwnerID: ownerID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: core.ProjectListDefaultSize, After: uuid.Max},
			daoMock:    &daoMock[[]*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockProjectListDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectList(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
		})
	}
}
