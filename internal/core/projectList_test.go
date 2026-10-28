package core_test

import (
	"errors"
	"testing"

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
		daoMock    *daoMock[[]*dao.ProjectListRow]

		expect    *core.ProjectPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectListRequest{OwnerID: ownerID, Limit: 10, Offset: 30},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 10, Offset: 30},
			daoMock:    &daoMock[[]*dao.ProjectListRow]{resp: []*dao.ProjectListRow{{Project: *daoProject(), Total: 31}}},

			expect: &core.ProjectPage{
				Projects: []*core.Project{
					{
						ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
						CreatedAt: createdAt, UpdatedAt: updatedAt,
					},
				},
				Total: 31,
			},
		},
		{
			name: "Success/Defaults",

			request: &core.ProjectListRequest{OwnerID: ownerID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: core.ProjectListDefaultSize},
			daoMock:    &daoMock[[]*dao.ProjectListRow]{},

			expect: &core.ProjectPage{Projects: []*core.Project{}},
		},
		{
			name: "Error/LimitTooHigh",

			request: &core.ProjectListRequest{OwnerID: ownerID, Limit: 101},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.ProjectListRequest{OwnerID: ownerID, Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.ProjectListRequest{OwnerID: ownerID, Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Internal",

			request: &core.ProjectListRequest{OwnerID: ownerID},

			daoRequest: &dao.ProjectListRequest{OwnerID: ownerID, Limit: core.ProjectListDefaultSize},
			daoMock:    &daoMock[[]*dao.ProjectListRow]{err: errFoo},

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
