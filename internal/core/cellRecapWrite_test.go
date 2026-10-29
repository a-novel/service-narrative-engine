package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// recapOutput is a consolidate output, as a recap write takes it.
var recapOutput = json.RawMessage(
	`{"condensed":"Marie attend.","compact":"Marie attend.","essentials":"Marie attend.","names":["Marie"]}`,
)

func TestCellRecapWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := func(recap json.RawMessage) *core.CellRecapWriteRequest {
		return &core.CellRecapWriteRequest{
			OwnerID: ownerID, ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1,
			Recap: recap, CellUpdatedAt: updatedAt,
		}
	}

	testCases := []struct {
		name string

		request *core.CellRecapWriteRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[struct{}]

		expectErr error
	}{
		{
			name: "Success",

			request: request(recapOutput),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
		},
		{
			name: "Error/NoStamp",

			request: &core.CellRecapWriteRequest{
				OwnerID: ownerID, ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1, Recap: recapOutput,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MalformedRecap",

			request: request(json.RawMessage(`{"condensed":"Marie attend."}`)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// A project of another user reads as absent, and nothing is written.
			name: "Error/OtherOwner",

			request: request(recapOutput),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CellNotFound",

			request: request(recapOutput),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{err: dao.ErrCellRecapUpdateNotFound},

			expectErr: core.ErrCellNotFound,
		},
		{
			name: "Error/Conflict",

			request: request(recapOutput),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{err: dao.ErrCellRecapUpdateConflict},

			expectErr: core.ErrRecapConflict,
		},
		{
			name: "Error/Owner",

			request: request(recapOutput),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Dao",

			request: request(recapOutput),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockCellRecapWriteDaoOwner(t)
			mockDao := coremocks.NewMockCellRecapWriteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CellRecapUpdateRequest{
						ProjectID: projectID, SceneID: sceneID1, LayerID: layerID1, Recap: recapOutput,
						CellUpdatedAt: updatedAt,
					}).
					Return(testCase.daoMock.err)
			}

			service := core.NewCellRecapWrite(mockOwner, mockDao)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockOwner.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
