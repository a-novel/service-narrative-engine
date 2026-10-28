package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
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
	notesCell := core.CellWriteItem{SceneID: sceneID1, LayerID: layerID2, Content: notes}
	manuscriptCell := core.CellWriteItem{SceneID: sceneID1, LayerID: layerID1, Content: manuscript}
	nextNotesCell := core.CellWriteItem{SceneID: sceneID2, LayerID: layerID2, Content: notes}
	notesKind := &dao.LayerKind{ID: layerID2, Kind: core.LayerKindNotes, Count: 1}
	manuscriptKind := &dao.LayerKind{ID: layerID1, Kind: core.LayerKindManuscript, Count: 1}

	request := func(cells ...core.CellWriteItem) *core.CellWriteRequest {
		return &core.CellWriteRequest{OwnerID: ownerID, ProjectID: projectID, Cells: cells}
	}

	daoCell := func(cell core.CellWriteItem) *dao.Cell {
		return &dao.Cell{
			ProjectID: projectID, SceneID: cell.SceneID, LayerID: cell.LayerID, Content: cell.Content,
			UpdatedAt: updatedAt,
		}
	}

	coreCell := func(cell core.CellWriteItem) *core.Cell {
		return &core.Cell{
			ProjectID: projectID, SceneID: cell.SceneID, LayerID: cell.LayerID, Content: cell.Content,
			UpdatedAt: updatedAt,
		}
	}

	testCases := []struct {
		name string

		request *core.CellWriteRequest

		ownerMock *daoMock[bool]
		kindMock  *daoMock[[]*dao.LayerKind]
		daoMock   *daoMock[[]*dao.Cell]

		expect    []*core.Cell
		expectErr error
	}{
		{
			name: "Success/Notes",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			daoMock:   &daoMock[[]*dao.Cell]{resp: []*dao.Cell{daoCell(notesCell)}},

			expect: []*core.Cell{coreCell(notesCell)},
		},
		{
			name: "Success/Manuscript",

			request: request(manuscriptCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{manuscriptKind}},
			daoMock:   &daoMock[[]*dao.Cell]{resp: []*dao.Cell{daoCell(manuscriptCell)}},

			expect: []*core.Cell{coreCell(manuscriptCell)},
		},
		{
			name: "Success/Batch",

			request: request(notesCell, manuscriptCell, nextNotesCell),

			ownerMock: &daoMock[bool]{resp: true},
			// Both reads return their rows in no particular order.
			kindMock: &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{manuscriptKind, notesKind}},
			daoMock: &daoMock[[]*dao.Cell]{resp: []*dao.Cell{
				daoCell(nextNotesCell), daoCell(manuscriptCell), daoCell(notesCell),
			}},

			expect: []*core.Cell{coreCell(notesCell), coreCell(manuscriptCell), coreCell(nextNotesCell)},
		},
		{
			name: "Error/MissingContent",

			request: request(core.CellWriteItem{SceneID: sceneID1, LayerID: layerID2}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooLarge",

			request: request(core.CellWriteItem{SceneID: sceneID1, LayerID: layerID2, Content: oversize}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoCell",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyCells",

			request: request(lo.Times(101, func(int) core.CellWriteItem {
				return core.CellWriteItem{SceneID: uuid.New(), LayerID: layerID2, Content: notes}
			})...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/DuplicateCell",

			request: request(notesCell, notesCell),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Kind",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/LayerNotFound",

			request: request(notesCell, manuscriptCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},

			expectErr: core.ErrLayerNotFound,
		},
		{
			// One cell holds notes on the manuscript layer, so no cell is written.
			name: "Error/ContentOfAnotherKind",

			request: request(notesCell, core.CellWriteItem{SceneID: sceneID1, LayerID: layerID1, Content: notes}),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind, manuscriptKind}},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SceneNotFound",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			daoMock:   &daoMock[[]*dao.Cell]{err: dao.ErrCellUpsertSceneNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/LayerDeletedMeanwhile",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			daoMock:   &daoMock[[]*dao.Cell]{err: dao.ErrCellUpsertLayerNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Internal",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			daoMock:   &daoMock[[]*dao.Cell]{err: errFoo},

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
				// One read names each layer once.
				layerIDs := lo.Uniq(lo.Map(testCase.request.Cells, func(cell core.CellWriteItem, _ int) uuid.UUID {
					return cell.LayerID
				}))

				mockKind.EXPECT().
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, IDs: layerIDs}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.CellUpsertRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.EqualFunc(
							req.Cells, testCase.request.Cells, func(item dao.CellUpsertItem, cell core.CellWriteItem) bool {
								return item.SceneID == cell.SceneID && item.LayerID == cell.LayerID &&
									bytes.Equal(item.Content, cell.Content)
							},
						)
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
