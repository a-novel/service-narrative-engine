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
	manuscript := json.RawMessage(`{"blocks":[{"kind":"narration","text":"Il pleut."}]}`)
	// oversize is a JSON string one byte over the 2 MiB a cell holds.
	oversize := json.RawMessage(`"` + strings.Repeat("a", 2<<20-1) + `"`)

	request := func(layerID uuid.UUID, content json.RawMessage) *core.CellWriteRequest {
		return &core.CellWriteRequest{
			OwnerID: ownerID, ProjectID: projectID, SceneID: sceneID1, LayerID: layerID, Content: content,
		}
	}

	testCases := []struct {
		name string

		request *core.CellWriteRequest

		ownerMock *daoMock[bool]
		layerMock *daoMock[*dao.Layer]
		daoMock   *daoMock[*dao.Cell]

		expect    *core.Cell
		expectErr error
	}{
		{
			name: "Success/Notes",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			daoMock: &daoMock[*dao.Cell]{resp: &dao.Cell{
				ProjectID: projectID, SceneID: sceneID1, LayerID: layerID2, Content: notes, UpdatedAt: updatedAt,
			}},

			expect: &core.Cell{
				ProjectID: projectID, SceneID: sceneID1, LayerID: layerID2, Content: notes, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Success/Manuscript",

			request: request(layerID1, manuscript),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},
			daoMock: &daoMock[*dao.Cell]{resp: &dao.Cell{
				ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1, Content: manuscript, UpdatedAt: updatedAt,
			}},

			expect: &core.Cell{
				ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1, Content: manuscript, UpdatedAt: updatedAt,
			},
		},
		{
			name: "Error/MissingContent",

			request: request(layerID2, nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooLarge",

			request: request(layerID2, oversize),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/LayerNotFound",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/ContentOfAnotherKind",

			request: request(layerID1, notes),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID1, 0, core.LayerKindManuscript)},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SceneNotFound",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			daoMock:   &daoMock[*dao.Cell]{err: dao.ErrCellUpsertSceneNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/LayerDeletedMeanwhile",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			daoMock:   &daoMock[*dao.Cell]{err: dao.ErrCellUpsertLayerNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Internal",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			layerMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			daoMock:   &daoMock[*dao.Cell]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockCellWriteDaoOwner(t)
			mockLayer := coremocks.NewMockCellWriteDaoLayer(t)
			mockDao := coremocks.NewMockCellWriteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.layerMock != nil {
				mockLayer.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ProjectID: projectID, ID: testCase.request.LayerID}).
					Return(testCase.layerMock.resp, testCase.layerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.CellUpsertRequest) bool {
						return req.ProjectID == projectID && req.SceneID == sceneID1 &&
							req.LayerID == testCase.request.LayerID &&
							string(req.Content) == string(testCase.request.Content) && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewCellWrite(mockOwner, mockLayer, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockLayer.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
