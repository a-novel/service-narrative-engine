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

		lockMock   *daoMock[*dao.Project]
		anchorMock *daoMock[*dao.Layer]
		// position is where the service inserts the layer, when the case reaches the insert.
		position   int
		insertMock *daoMock[*dao.Layer]
		transactor transaction.Transactor

		expect    *core.Layer
		expectErr error
	}{
		{
			name: "Success/First",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			position:   0,
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 0)},

			expect: coreLayer(layerID3, 0),
		},
		{
			name: "Success/AfterAnchor",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer", After: layerID2},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID2, 1)},
			position:   2,
			insertMock: &daoMock[*dao.Layer]{resp: notesLayer(layerID3, 2)},

			expect: coreLayer(layerID3, 2),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock: &daoMock[*dao.Project]{err: dao.ErrProjectLockNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock: &daoMock[*dao.Project]{resp: foreignProject()},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/UnknownAnchor",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer", After: layerID2},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{err: dao.ErrLayerSelectNotFound},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/AnchorElsewhere",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer", After: layerID2},

			lockMock: &daoMock[*dao.Project]{resp: daoProject()},
			anchorMock: &daoMock[*dao.Layer]{resp: func() *dao.Layer {
				anchor := notesLayer(layerID3, 2)
				anchor.ProjectID = otherProjectID

				return anchor
			}()},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectFull",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[*dao.Layer]{err: dao.ErrLayerInsertFull},

			expectErr: core.ErrProjectFull,
		},
		{
			name: "Error/Insert",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			lockMock:   &daoMock[*dao.Project]{resp: daoProject()},
			insertMock: &daoMock[*dao.Layer]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: &core.LayerCreateRequest{OwnerID: ownerID, ProjectID: projectID, Name: "Layer"},

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockLayerCreateDaoLock(t)
			mockSelect := coremocks.NewMockLayerCreateDaoSelect(t)
			mockInsert := coremocks.NewMockLayerCreateDaoInsert(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, &dao.ProjectLockRequest{ID: projectID}).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.anchorMock != nil {
				mockSelect.EXPECT().
					Exec(mock.Anything, &dao.LayerSelectRequest{ID: testCase.request.After}).
					Return(testCase.anchorMock.resp, testCase.anchorMock.err)
			}

			if testCase.insertMock != nil {
				mockInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.LayerInsertRequest) bool {
						return req.ProjectID == projectID && req.Position == testCase.position &&
							req.Kind == core.LayerKindNotes && req.Name == "Layer" && recent(req.Now)
					})).
					Return(testCase.insertMock.resp, testCase.insertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewLayerCreate(mockLock, mockSelect, mockInsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockSelect.AssertExpectations(t)
			mockInsert.AssertExpectations(t)
		})
	}
}
