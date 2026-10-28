package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestLayerCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(position int, kind string) *core.LayerCreateRequest {
		return &core.LayerCreateRequest{
			OwnerID: ownerID, ProjectID: projectID, Position: position, Kind: kind, Name: "Layer",
			Description: "Who appears.",
		}
	}

	// manuscript is a second manuscript layer, which a project may hold.
	manuscript := coreLayer(layerID3, 1)
	manuscript.Kind = core.LayerKindManuscript

	testCases := []struct {
		name string

		request *core.LayerCreateRequest

		lockMock   *daoMock[bool]
		insertMock *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: request(0, core.LayerKindNotes),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 0)},

			expect: coreLayer(layerID3, 0),
		},
		{
			name: "Success/Position",

			request: request(2, core.LayerKindNotes),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},

			expect: coreLayer(layerID3, 2),
		},
		{
			name: "Success/Manuscript",

			request: request(1, core.LayerKindManuscript),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Layer]{resp: daoLayer(layerID3, 1, core.LayerKindManuscript)},

			expect: manuscript,
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Kind: core.LayerKindNotes},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownKind",

			request: request(0, "chapter"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: request(-1, core.LayerKindNotes),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(0, core.LayerKindNotes),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(0, core.LayerKindNotes),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: request(0, core.LayerKindNotes),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Layer]{err: dao.ErrLayerInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: request(0, core.LayerKindNotes),

			lockMock:   &daoMock[bool]{resp: true},
			insertMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(0, core.LayerKindNotes),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockLayerCreateDaoLock(t)
			mockInsert := coremocks.NewMockLayerCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						return req.ID != uuid.Nil && req.ProjectID == projectID && req.Position == testCase.request.Position &&
							req.Kind == testCase.request.Kind && req.Name == "Layer" && req.Description == "Who appears." &&
							recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerCreate(mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
