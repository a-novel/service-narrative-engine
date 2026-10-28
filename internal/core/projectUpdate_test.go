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

func TestProjectUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectUpdateRequest{OwnerID: ownerID, ID: projectID, Title: "Clear Hours", Language: "en"}

	testCases := []struct {
		name string

		request *core.ProjectUpdateRequest

		selectMock *daoMock[*dao.Project]
		daoMock    *daoMock[*dao.Project]

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			selectMock: &daoMock[*dao.Project]{resp: daoProject()},
			daoMock:    &daoMock[*dao.Project]{resp: daoProject()},

			expect: &core.Project{
				ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/InvalidRequest",

			request: &core.ProjectUpdateRequest{OwnerID: ownerID, ID: projectID, Language: "en"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: request,

			selectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: request,

			selectMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/DeletedMeanwhile",

			request: request,

			selectMock: &daoMock[*dao.Project]{resp: daoProject()},
			daoMock:    &daoMock[*dao.Project]{err: dao.ErrProjectUpdateNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Internal",

			request: request,

			selectMock: &daoMock[*dao.Project]{resp: daoProject()},
			daoMock:    &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSelect := coremocks.NewMockProjectUpdateDaoSelect(t)
			mockDao := coremocks.NewMockProjectUpdateDao(t)

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectUpdateRequest) bool {
						return req.ID == projectID && req.Title == "Clear Hours" && req.Language == "en" && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectUpdate(mockSelect, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockSelect.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
