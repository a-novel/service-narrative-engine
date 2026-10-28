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
		countMock  *daoMock[*dao.ProjectCount]

		expect    *core.ProjectPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectListRequest{OwnerID: ownerID, Limit: 10, After: projectID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 10, After: projectID},
			daoMock:    &daoMock[[]*dao.Project]{resp: []*dao.Project{daoProject()}},
			countMock:  &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{Count: 3}},

			expect: &core.ProjectPage{
				Projects: []*core.Project{
					{
						ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
						CreatedAt: createdAt, UpdatedAt: updatedAt,
					},
				},
				Total: 3,
			},
		},
		{
			name: "Success/Defaults",

			request: &core.ProjectListRequest{OwnerID: ownerID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: core.ProjectListDefaultSize, After: uuid.Max},
			daoMock:    &daoMock[[]*dao.Project]{resp: []*dao.Project{}},
			countMock:  &daoMock[*dao.ProjectCount]{resp: &dao.ProjectCount{}},

			expect: &core.ProjectPage{Projects: []*core.Project{}},
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
		{
			name: "Error/Count",

			request: &core.ProjectListRequest{OwnerID: ownerID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: core.ProjectListDefaultSize, After: uuid.Max},
			daoMock:    &daoMock[[]*dao.Project]{resp: []*dao.Project{}},
			countMock:  &daoMock[*dao.ProjectCount]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockProjectListDao(t)
			mockCount := coremocks.NewMockProjectListDaoCount(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.countMock != nil {
				mockCount.EXPECT().
					Exec(mock.Anything, &dao.ProjectCountRequest{OwnerID: ownerID}).
					Return(testCase.countMock.resp, testCase.countMock.err)
			}

			service := core.NewProjectList(mockDao, mockCount)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
			mockCount.AssertExpectations(t)
		})
	}
}
