package core_test

import (
	"context"
	"errors"
	"slices"
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

func TestScenesCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	scene := core.ScenesCreateItem{BeatID: beatID1, Title: "Scene"}
	other := core.ScenesCreateItem{
		BeatID: beatID2, Position: 2, Title: "Le thème", Description: "Paul ment.", Prompt: "Lent.",
	}

	request := func(scenes ...core.ScenesCreateItem) *core.ScenesCreateRequest {
		return &core.ScenesCreateRequest{OwnerID: ownerID, ProjectID: projectID, Scenes: scenes}
	}

	// insertItem is a scene as the insert receives it, without its generated ID.
	insertItem := func(item core.ScenesCreateItem) dao.ScenesInsertItem {
		return dao.ScenesInsertItem{
			BeatID: item.BeatID, Position: item.Position, Title: item.Title, Description: item.Description,
			Prompt: item.Prompt,
		}
	}

	// stored is a scene as the service returns it, without its generated ID. The fake insert places it at
	// the position it was asked for.
	stored := func(item core.ScenesCreateItem) *core.Scene {
		return &core.Scene{
			ProjectID: projectID, BeatID: item.BeatID, Position: item.Position, Title: item.Title,
			Description: item.Description, Prompt: item.Prompt, UpdatedAt: updatedAt,
		}
	}

	testCases := []struct {
		name string

		request *core.ScenesCreateRequest

		lockMock *daoMock[bool]
		// insert reports whether the service runs the insert, which fails with insertErr when set.
		insert     bool
		insertErr  error
		transactor transaction.Transactor

		expect    []*core.Scene
		expectErr error
	}{
		{
			name: "Success",

			request: request(scene),

			lockMock: &daoMock[bool]{resp: true},
			insert:   true,

			expect: []*core.Scene{stored(scene)},
		},
		{
			name: "Success/AcrossBeats",

			request: request(scene, other),

			lockMock: &daoMock[bool]{resp: true},
			insert:   true,

			// The insert returns its rows in no particular order, and the service returns the request's.
			expect: []*core.Scene{stored(scene), stored(other)},
		},
		{
			name: "Error/InvalidRequest",

			request: request(core.ScenesCreateItem{BeatID: beatID1, Description: "Paul ment."}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingBeat",

			request: request(core.ScenesCreateItem{Title: "Scene"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(core.ScenesCreateItem{BeatID: beatID1, Position: -1, Title: "Scene"}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoScene",

			request: request(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyScenes",

			request: request(slices.Repeat([]core.ScenesCreateItem{scene}, 101)...),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(scene),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(scene),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/BeatFull",

			request: request(scene),

			lockMock:  &daoMock[bool]{resp: true},
			insert:    true,
			insertErr: dao.ErrScenesInsertFull,

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/BeatNotFound",

			request: request(scene, other),

			lockMock:  &daoMock[bool]{resp: true},
			insert:    true,
			insertErr: dao.ErrScenesInsertBeatNotFound,

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Insert",

			request: request(scene),

			lockMock:  &daoMock[bool]{resp: true},
			insert:    true,
			insertErr: errFoo,

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(scene),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockScenesCreateDaoLock(t)
			mockInsert := coremocks.NewMockScenesCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			// insertedIDs are the IDs the service generated for the insert, in the order of the request.
			var insertedIDs []uuid.UUID

			if testCase.insert {
				insertItems := lo.Map(testCase.request.Scenes, func(item core.ScenesCreateItem, _ int) dao.ScenesInsertItem {
					return insertItem(item)
				})

				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ScenesInsertRequest) bool {
						items := lo.Map(req.Scenes, func(item dao.ScenesInsertItem, _ int) dao.ScenesInsertItem {
							item.ID = uuid.Nil

							return item
						})

						return req.ProjectID == projectID && recent(req.Now) && slices.Equal(items, insertItems)
					})).
					RunAndReturn(func(_ context.Context, req *dao.ScenesInsertRequest) ([]*dao.Scene, error) {
						insertedIDs = lo.Map(req.Scenes, func(item dao.ScenesInsertItem, _ int) uuid.UUID { return item.ID })

						if testCase.insertErr != nil {
							return nil, testCase.insertErr
						}

						rows := lo.Map(req.Scenes, func(item dao.ScenesInsertItem, _ int) *dao.Scene {
							return &dao.Scene{
								ID: item.ID, ProjectID: projectID, BeatID: item.BeatID, Position: item.Position,
								Title: item.Title, Description: item.Description, Prompt: item.Prompt, UpdatedAt: updatedAt,
							}
						})

						slices.Reverse(rows)

						return rows, nil
					})
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewScenesCreate(mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expect != nil {
				require.True(t, freshIDs(insertedIDs))
				require.Equal(t, insertedIDs, lo.Map(result, func(scene *core.Scene, _ int) uuid.UUID { return scene.ID }))

				for _, scene := range result {
					scene.ID = uuid.Nil
				}
			}

			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
