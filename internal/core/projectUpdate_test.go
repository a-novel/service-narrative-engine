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

	updated := daoProject()
	updated.Title = "Clear Hours"
	updated.Language = "en"
	updated.Revision = 2

	validRequest := &core.ProjectUpdateRequest{
		OwnerID: ownerID, ID: projectID, Revision: 1, Title: "Clear Hours", Language: "en",
	}

	testCases := []struct {
		name string

		request *core.ProjectUpdateRequest

		daoMock    *daoMock[*dao.Project]
		selectMock *daoMock[*dao.Project]

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			daoMock: &daoMock[*dao.Project]{resp: updated},

			expect: &core.Project{
				ID: projectID, OwnerID: ownerID, Title: "Clear Hours", Language: "en",
				Revision: 2, CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingRevision",

			request: &core.ProjectUpdateRequest{OwnerID: ownerID, ID: projectID, Title: "Clear Hours", Language: "en"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RevisionConflict",

			request: validRequest,

			daoMock:    &daoMock[*dao.Project]{err: dao.ErrProjectUpdateNotFound},
			selectMock: &daoMock[*dao.Project]{resp: daoProject()},

			expectErr: core.ErrRevisionConflict,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			daoMock:    &daoMock[*dao.Project]{err: dao.ErrProjectUpdateNotFound},
			selectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Select",

			request: validRequest,

			daoMock:    &daoMock[*dao.Project]{err: dao.ErrProjectUpdateNotFound},
			selectMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			daoMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockProjectUpdateDao(t)
			mockSelect := coremocks.NewMockProjectUpdateDaoSelect(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectUpdateRequest) bool {
						return req.ID == projectID && req.OwnerID == ownerID && req.Revision == 1 &&
							req.Title == "Clear Hours" && req.Language == "en" && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			service := core.NewProjectUpdate(mockDao, mockSelect)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
		})
	}
}
