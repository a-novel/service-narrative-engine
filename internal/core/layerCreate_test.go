package core_test

import (
	"errors"
	"testing"

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

	testCases := []struct {
		name string

		request *core.LayerCreateRequest

		ownerMock  *daoMock[bool]
		lockMock   *daoMock[*dao.Project]
		insertMock *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success/First",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 0)},

			expect: coreLayer(layerID3, 0),
		},
		{
			name: "Success/Position",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer", Position: 2},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},

			expect: coreLayer(layerID3, 2),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativePosition",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer", Position: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Lock",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock: &daoMock[bool]{resp: true},
			lockMock:  &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectFull",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[*dao.Layer]{err: dao.ErrLayerInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock:  &daoMock[bool]{resp: true},
			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			ownerMock:  &daoMock[bool]{resp: true},
			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockLayerCreateDaoOwner(t)
			mockLock := coremocks.NewMockLayerCreateDaoLock(t)
			mockInsert := coremocks.NewMockLayerCreateDaoInsert(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						return req.ProjectID == projectID && req.Position == testCase.request.Position &&
							req.Kind == core.LayerKindNotes && req.Name == "Layer" && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerCreate(mockOwner, mockLock, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockLock.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
