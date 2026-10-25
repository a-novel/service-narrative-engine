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

func TestSceneUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	updated := daoScene(sceneID1, 0)
	updated.Synopsis = "Paul ment."
	updated.Revision = 2

	validRequest := &core.SceneUpdateRequest{
		OwnerID: ownerID, ID: sceneID1, Revision: 1, Title: "Scene", Synopsis: "Paul ment.",
	}

	testCases := []struct {
		name string

		request *core.SceneUpdateRequest

		daoMock    *daoMock[*dao.Scene]
		selectMock *daoMock[*dao.Scene]

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			daoMock: &daoMock[*dao.Scene]{resp: updated},

			expect: &core.Scene{
				ID: sceneID1, ProjectID: projectID, Title: "Scene", Synopsis: "Paul ment.", Revision: 2,
				UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/SynopsisTooLong",

			request: &core.SceneUpdateRequest{OwnerID: ownerID, ID: sceneID1, Revision: 1, Synopsis: string(make([]byte, 4001))},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RevisionConflict",

			request: validRequest,

			daoMock:    &daoMock[*dao.Scene]{err: dao.ErrSceneUpdateNotFound},
			selectMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},

			expectErr: core.ErrRevisionConflict,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			daoMock:    &daoMock[*dao.Scene]{err: dao.ErrSceneUpdateNotFound},
			selectMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/Internal",

			request: validRequest,

			daoMock: &daoMock[*dao.Scene]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockSceneUpdateDao(t)
			mockSelect := coremocks.NewMockSceneUpdateDaoSelect(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneUpdateRequest) bool {
						return req.ID == sceneID1 && req.OwnerID == ownerID && req.Revision == 1 &&
							req.Synopsis == "Paul ment." && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.selectMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: sceneID1, OwnerID: ownerID}).
					Return(testCase.selectMock.resp, testCase.selectMock.err)
			}

			service := core.NewSceneUpdate(mockDao, mockSelect)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
		})
	}
}
