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
	updated.Title = "Ouverture"
	updated.Color = "#e0457b"
	updated.Description = "Marie attend."
	updated.Prompt = "Garder le suspense."

	validRequest := &core.SceneUpdateRequest{
		OwnerID: ownerID, ID: sceneID1, Title: "Ouverture", Color: "#E0457B",
		Description: "Marie attend.", Prompt: "Garder le suspense.",
	}

	testCases := []struct {
		name string

		request *core.SceneUpdateRequest

		daoMock *daoMock[*dao.Scene]

		expect    *core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: validRequest,

			daoMock: &daoMock[*dao.Scene]{resp: updated},

			expect: &core.Scene{
				ID: sceneID1, ProjectID: projectID, Title: "Ouverture", Color: "#e0457b",
				Description: "Marie attend.", Prompt: "Garder le suspense.", UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/InvalidColor",

			request: &core.SceneUpdateRequest{OwnerID: ownerID, ID: sceneID1, Title: "Ouverture", Color: "#e04"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: validRequest,

			daoMock: &daoMock[*dao.Scene]{err: dao.ErrSceneUpdateNotFound},

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

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.SceneUpdateRequest) bool {
						// The color is stored in lowercase.
						return req.ID == sceneID1 && req.OwnerID == ownerID && req.Title == "Ouverture" &&
							req.Color == "#e0457b" && req.Description == "Marie attend." &&
							req.Prompt == "Garder le suspense." && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewSceneUpdate(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
		})
	}
}
