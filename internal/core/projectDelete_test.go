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

func TestProjectDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	validRequest := &core.ProjectDeleteRequest{OwnerID: ownerID, ID: projectID, Revision: 1}

	testCases := []struct {
		name string

		request *core.ProjectDeleteRequest

		daoMock    *daoMock[*dao.Project]
		selectMock *daoMock[*dao.Project]

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			daoMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: &core.Project{
				ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Language: "fr",
				Revision: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingID",

			request: &core.ProjectDeleteRequest{OwnerID: ownerID, Revision: 1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RevisionConflict",

			request: validRequest,

			daoMock:    &daoMock[*dao.Project]{err: dao.ErrProjectDeleteNotFound},
			selectMock: &daoMock[*dao.Project]{resp: daoProject()},

			expectErr: core.ErrRevisionConflict,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			daoMock:    &daoMock[*dao.Project]{err: dao.ErrProjectDeleteNotFound},
			selectMock: &daoMock[*dao.Project]{err: dao.ErrProjectSelectNotFound},

			expectErr: core.ErrProjectNotFound,
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

			mockDao := coremocks.NewMockProjectDeleteDao(t)
			mockSelect := coremocks.NewMockProjectDeleteDaoSelect(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.ProjectDeleteRequest{ID: projectID, OwnerID: ownerID, Revision: 1}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			service := core.NewProjectDelete(mockDao, mockSelect)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
		})
	}
}
