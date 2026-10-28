package core_test

import (
	"bytes"
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

	// layerID2 holds notes and layerID1 the manuscript.
	notesKind := []*dao.LayerKind{{ID: layerID2, Kind: core.LayerKindNotes, Count: 1}}
	manuscriptKind := []*dao.LayerKind{{ID: layerID1, Kind: core.LayerKindManuscript, Count: 1}}

	request := func(layerID uuid.UUID, content json.RawMessage) *core.CellWriteRequest {
		return &core.CellWriteRequest{
			OwnerID: ownerID, ProjectID: projectID, SceneID: sceneID1, LayerID: layerID, Content: content,
		}
	}

	daoRow := func(layerID uuid.UUID) *dao.CellUpsertRow {
		return &dao.CellUpsertRow{SceneID: sceneID1, LayerID: layerID, UpdatedAt: updatedAt}
	}

	writtenCell := func(layerID uuid.UUID) *core.WrittenCell {
		return &core.WrittenCell{SceneID: sceneID1, LayerID: layerID, UpdatedAt: updatedAt}
	}

	testCases := []struct {
		name string

		request *core.CellWriteRequest

		ownerMock *daoMock[bool]
		kindMock  *daoMock[[]*dao.LayerKind]
		daoMock   *daoMock[*dao.CellUpsertRow]

		expect    *core.WrittenCell
		expectErr error
	}{
		{
			name: "Success/Notes",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: notesKind},
			daoMock:   &daoMock[*dao.CellUpsertRow]{resp: daoRow(layerID2)},

			expect: writtenCell(layerID2),
		},
		{
			name: "Success/Manuscript",

			request: request(layerID1, manuscript),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: manuscriptKind},
			daoMock:   &daoMock[*dao.CellUpsertRow]{resp: daoRow(layerID1)},

			expect: writtenCell(layerID1),
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
			name: "Error/Kind",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/LayerNotFound",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{}},

			expectErr: core.ErrLayerNotFound,
		},
		{
			// Notes do not follow the manuscript schema.
			name: "Error/ContentOfAnotherKind",

			request: request(layerID1, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: manuscriptKind},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SceneNotFound",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: notesKind},
			daoMock:   &daoMock[*dao.CellUpsertRow]{err: dao.ErrCellUpsertSceneNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/LayerDeletedMeanwhile",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: notesKind},
			daoMock:   &daoMock[*dao.CellUpsertRow]{err: dao.ErrCellUpsertLayerNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Internal",

			request: request(layerID2, notes),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: notesKind},
			daoMock:   &daoMock[*dao.CellUpsertRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockCellWriteDaoOwner(t)
			mockKind := coremocks.NewMockCellWriteDaoKind(t)
			mockDao := coremocks.NewMockCellWriteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.kindMock != nil {
				mockKind.EXPECT().
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{testCase.request.LayerID}}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.CellUpsertRequest) bool {
						return req.ProjectID == projectID && req.SceneID == sceneID1 && req.LayerID == testCase.request.LayerID &&
							bytes.Equal(req.Content, testCase.request.Content) && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewCellWrite(mockOwner, mockKind, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockKind.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
