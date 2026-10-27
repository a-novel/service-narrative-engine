package core_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestCellWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	notes := json.RawMessage(`{"text":"Marie"}`)
	manuscript := json.RawMessage(`{"blocks":[{"id":"b1","kind":"narration","text":"Il pleut."}]}`)
	duplicates := json.RawMessage(
		`{"blocks":[{"id":"b1","kind":"narration","text":""},{"id":"b1","kind":"narration","text":""}]}`,
	)

	otherProjectLayer := notesLayer(layerID2, 0)
	otherProjectLayer.ProjectID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

	request := func(layerID uuid.UUID, content json.RawMessage) *core.CellWriteRequest {
		return &core.CellWriteRequest{
			OwnerID: ownerID, SceneID: sceneID1, LayerID: layerID, Content: content,
		}
	}

	testCases := []struct {
		name string

		request *core.CellWriteRequest

		sceneMock *daoMock[*dao.Scene]
		layerMock *daoMock[*dao.Layer]
		daoMock   *daoMock[*dao.Cell]

		expect    *core.Cell
		expectErr error
	}{
		{
			name: "Success/Notes",

			request: request(layerID2, notes),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			daoMock: &daoMock[*dao.Cell]{resp: &dao.Cell{
				SceneID: sceneID1, LayerID: layerID2, Content: notes, UpdatedAt: updatedAt,
			}},

			expect: &core.Cell{SceneID: sceneID1, LayerID: layerID2, Content: notes, UpdatedAt: updatedAt},
		},
		{
			name: "Success/Manuscript",

			request: request(layerID1, manuscript),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},
			daoMock: &daoMock[*dao.Cell]{resp: &dao.Cell{
				SceneID: sceneID1, LayerID: layerID1, Content: manuscript, UpdatedAt: updatedAt,
			}},

			expect: &core.Cell{SceneID: sceneID1, LayerID: layerID1, Content: manuscript, UpdatedAt: updatedAt},
		},
		{
			name: "Error/MissingContent",

			request: request(layerID2, nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooLarge",

			request: request(layerID2, json.RawMessage(`{"text":"`+strings.Repeat("a", 128)+`"}`)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SceneNotFound",

			request: request(layerID2, notes),

			sceneMock: &daoMock[*dao.Scene]{err: dao.ErrSceneSelectNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/LayerNotFound",

			request: request(layerID2, notes),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/OtherProject",

			request: request(layerID2, notes),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{resp: otherProjectLayer},

			expectErr: core.ErrCellNotFound,
		},
		{
			name: "Error/ContentOfAnotherKind",

			request: request(layerID1, notes),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateBlockID",

			request: request(layerID1, duplicates),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Internal",

			request: request(layerID2, notes),

			sceneMock: &daoMock[*dao.Scene]{resp: daoScene(sceneID1, 0)},
			layerMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			daoMock:   &daoMock[*dao.Cell]{err: errFoo},

			expectErr: errFoo,
		},
	}

	limits := testLimits
	limits.MaxCellBytes = 100

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockScene := coremocks.NewMockCellWriteDaoScene(t)
			mockLayer := coremocks.NewMockCellWriteDaoLayer(t)
			mockDao := coremocks.NewMockCellWriteDao(t)

			if testCase.sceneMock != nil {
				mockScene.EXPECT().
					Exec(mock.Anything, &dao.SceneSelectRequest{ID: sceneID1, OwnerID: ownerID}).
					Return(testCase.sceneMock.resp, testCase.sceneMock.err)
			}

			if testCase.layerMock != nil {
				mockLayer.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: testCase.request.LayerID, OwnerID: ownerID}).
					Return(testCase.layerMock.resp, testCase.layerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.CellUpsertRequest) bool {
						return req.SceneID == sceneID1 && req.LayerID == testCase.request.LayerID &&
							string(req.Content) == string(testCase.request.Content) && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewCellWrite(mockScene, mockLayer, mockDao, limits)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockScene.AssertExpectations(t)
			mockLayer.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
