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

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestCellsWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	notes := json.RawMessage(`{"text":"Marie"}`)
	manuscript := json.RawMessage(`{"blocks":[{"kind":"narration","text":"Il pleut."}]}`)
	// oversize is a JSON string one byte over the 2 MiB a cell holds.
	oversize := json.RawMessage(`"` + strings.Repeat("a", 2<<20-1) + `"`)

	// layerID2 holds notes and layerID1 the manuscript.
	notesCell := core.CellsWriteItem{SceneID: sceneID1, LayerID: layerID2, Content: notes}
	manuscriptCell := core.CellsWriteItem{SceneID: sceneID1, LayerID: layerID1, Content: manuscript}
	nextNotesCell := core.CellsWriteItem{SceneID: sceneID2, LayerID: layerID2, Content: notes}
	notesKind := &dao.LayerKind{ID: layerID2, Kind: core.LayerKindNotes, Count: 1}
	manuscriptKind := &dao.LayerKind{ID: layerID1, Kind: core.LayerKindManuscript, Count: 1}

	request := func(cells ...core.CellsWriteItem) *core.CellsWriteRequest {
		return &core.CellsWriteRequest{OwnerID: ownerID, ProjectID: projectID, Cells: cells}
	}

	daoRow := func(cell core.CellsWriteItem) *dao.CellUpsertRow {
		return &dao.CellUpsertRow{SceneID: cell.SceneID, LayerID: cell.LayerID, UpdatedAt: updatedAt}
	}

	writtenCell := func(cell core.CellsWriteItem) *core.WrittenCell {
		return &core.WrittenCell{SceneID: cell.SceneID, LayerID: cell.LayerID, UpdatedAt: updatedAt}
	}

	// sceneLengths are the lengths of the first two scenes once the cells are written.
	sceneLengths := func(first, second int) *daoMock[[]*dao.SceneLength] {
		return &daoMock[[]*dao.SceneLength]{resp: []*dao.SceneLength{
			{SceneID: sceneID1, Length: first}, {SceneID: sceneID2, Length: second},
		}}
	}

	testCases := []struct {
		name string

		request *core.CellsWriteRequest

		ownerMock  *daoMock[bool]
		kindMock   *daoMock[[]*dao.LayerKind]
		lockMock   *daoMock[struct{}]
		daoMock    *daoMock[[]*dao.CellUpsertRow]
		lengthMock *daoMock[[]*dao.SceneLength]
		transactor transaction.Transactor

		expect    []*core.WrittenCell
		expectErr error
	}{
		{
			name: "Success/Notes",

			request: request(notesCell),

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:   &daoMock[struct{}]{},
			daoMock:    &daoMock[[]*dao.CellUpsertRow]{resp: []*dao.CellUpsertRow{daoRow(notesCell)}},
			lengthMock: &daoMock[[]*dao.SceneLength]{resp: []*dao.SceneLength{{SceneID: sceneID1, Length: 100}}},

			expect: []*core.WrittenCell{writtenCell(notesCell)},
		},
		{
			name: "Success/Manuscript",

			request: request(manuscriptCell),

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{manuscriptKind}},
			lockMock:   &daoMock[struct{}]{},
			daoMock:    &daoMock[[]*dao.CellUpsertRow]{resp: []*dao.CellUpsertRow{daoRow(manuscriptCell)}},
			lengthMock: &daoMock[[]*dao.SceneLength]{resp: []*dao.SceneLength{{SceneID: sceneID1, Length: 100}}},

			expect: []*core.WrittenCell{writtenCell(manuscriptCell)},
		},
		{
			name: "Success/Batch",

			request: request(notesCell, manuscriptCell, nextNotesCell),

			ownerMock: &daoMock[bool]{resp: true},
			// The kind read and the write return their rows in no particular order.
			kindMock: &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{manuscriptKind, notesKind}},
			lockMock: &daoMock[struct{}]{},
			daoMock: &daoMock[[]*dao.CellUpsertRow]{resp: []*dao.CellUpsertRow{
				daoRow(nextNotesCell), daoRow(manuscriptCell), daoRow(notesCell),
			}},
			lengthMock: sceneLengths(100, 100),

			expect: []*core.WrittenCell{writtenCell(notesCell), writtenCell(manuscriptCell), writtenCell(nextNotesCell)},
		},
		{
			name: "Success/AtSceneContentMax",

			request: request(notesCell, nextNotesCell),

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:   &daoMock[struct{}]{},
			daoMock:    &daoMock[[]*dao.CellUpsertRow]{resp: []*dao.CellUpsertRow{daoRow(notesCell), daoRow(nextNotesCell)}},
			lengthMock: sceneLengths(core.SceneContentMax, core.SceneContentMax),

			expect: []*core.WrittenCell{writtenCell(notesCell), writtenCell(nextNotesCell)},
		},
		{
			name: "Error/MissingContent",

			request: request(core.CellsWriteItem{SceneID: sceneID1, LayerID: layerID2}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooLarge",

			request: request(core.CellsWriteItem{SceneID: sceneID1, LayerID: layerID2, Content: oversize}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoCell",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyCells",

			request: request(lo.Times(101, func(int) core.CellsWriteItem {
				return core.CellsWriteItem{SceneID: uuid.New(), LayerID: layerID2, Content: notes}
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

			request: request(notesCell, core.CellsWriteItem{SceneID: sceneID1, LayerID: layerID1, Content: notes}),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind, manuscriptKind}},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/SceneNotFound",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:  &daoMock[struct{}]{},
			daoMock:   &daoMock[[]*dao.CellUpsertRow]{err: dao.ErrCellsUpsertSceneNotFound},

			expectErr: core.ErrSceneNotFound,
		},
		{
			name: "Error/LayerDeletedMeanwhile",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:  &daoMock[struct{}]{},
			daoMock:   &daoMock[[]*dao.CellUpsertRow]{err: dao.ErrCellsUpsertLayerNotFound},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/Internal",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:  &daoMock[struct{}]{},
			daoMock:   &daoMock[[]*dao.CellUpsertRow]{err: errFoo},

			expectErr: errFoo,
		},
		{
			// The second scene would pass the cap, so neither cell is kept.
			name: "Error/SceneContentFull",

			request: request(notesCell, nextNotesCell),

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:   &daoMock[struct{}]{},
			daoMock:    &daoMock[[]*dao.CellUpsertRow]{resp: []*dao.CellUpsertRow{daoRow(notesCell), daoRow(nextNotesCell)}},
			lengthMock: sceneLengths(100, core.SceneContentMax+1),

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/LockScenes",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:  &daoMock[struct{}]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/SceneLength",

			request: request(notesCell),

			ownerMock:  &daoMock[bool]{resp: true},
			kindMock:   &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},
			lockMock:   &daoMock[struct{}]{},
			daoMock:    &daoMock[[]*dao.CellUpsertRow]{resp: []*dao.CellUpsertRow{daoRow(notesCell)}},
			lengthMock: &daoMock[[]*dao.SceneLength]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(notesCell),

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{notesKind}},

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockCellsWriteDaoOwner(t)
			mockKind := coremocks.NewMockCellsWriteDaoKind(t)
			mockLock := coremocks.NewMockCellsWriteDaoLock(t)
			mockDao := coremocks.NewMockCellsWriteDao(t)
			mockLength := coremocks.NewMockCellsWriteDaoLength(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.kindMock != nil {
				// One read names each layer once.
				layerIDs := lo.Uniq(lo.Map(testCase.request.Cells, func(cell core.CellsWriteItem, _ int) uuid.UUID {
					return cell.LayerID
				}))

				mockKind.EXPECT().
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, IDs: layerIDs}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
			}

			// The lock and the length name each scene once.
			sceneIDs := lo.Uniq(lo.Map(testCase.request.Cells, func(cell core.CellsWriteItem, _ int) uuid.UUID {
				return cell.SceneID
			}))

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.SceneLockRequest{ProjectID: projectID, IDs: sceneIDs}).
					Return(testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.CellsUpsertRequest) bool {
						return req.ProjectID == projectID && recent(req.Now) && slices.EqualFunc(
							req.Cells, testCase.request.Cells, func(item dao.CellsUpsertItem, cell core.CellsWriteItem) bool {
								return item.SceneID == cell.SceneID && item.LayerID == cell.LayerID &&
									bytes.Equal(item.Content, cell.Content)
							},
						)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.lengthMock != nil {
				mockLength.EXPECT().
					Exec(mock.Anything, &dao.SceneLengthRequest{ProjectID: projectID, IDs: sceneIDs}).
					Return(testCase.lengthMock.resp, testCase.lengthMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewCellsWrite(mockOwner, mockKind, mockLock, mockDao, mockLength, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockKind.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockDao.AssertExpectations(t)
			mockLength.AssertExpectations(t)
		})
	}
}
