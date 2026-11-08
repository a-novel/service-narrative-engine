package core_test

import (
	"encoding/json"
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

func TestBeatRecapWrite(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	sources := []*core.BeatRecapSource{{SceneID: sceneID1, CellUpdatedAt: updatedAt}, {SceneID: sceneID2}}
	storedSources := json.RawMessage(
		`[{"sceneID":"` + sceneID1.String() + `","cellUpdatedAt":"` + updatedAt.Format("2006-01-02T15:04:05Z07:00") +
			`"},{"sceneID":"` + sceneID2.String() + `","cellUpdatedAt":"0001-01-01T00:00:00Z"}]`,
	)

	request := func(sources []*core.BeatRecapSource) *core.BeatRecapWriteRequest {
		return &core.BeatRecapWriteRequest{
			OwnerID: ownerID, ProjectID: projectID, BeatID: beatID1, Recap: recapOutput, BeatUpdatedAt: updatedAt,
			Sources: sources,
		}
	}

	testCases := []struct {
		name string

		request *core.BeatRecapWriteRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[struct{}]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expectErr error
	}{
		{
			name: "Success",

			request: request(sources),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
			bytesMock: withinBudget,
		},
		{
			name: "Error/RepeatedScene",

			request: request([]*core.BeatRecapSource{{SceneID: sceneID1}, {SceneID: sceneID1}}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// A project of another user reads as absent, and nothing is written.
			name: "Error/OtherOwner",

			request: request(sources),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/BeatNotFound",

			request: request(sources),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[struct{}]{err: dao.ErrBeatRecapUpdateNotFound},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Conflict",

			request: request(sources),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[struct{}]{err: dao.ErrBeatRecapUpdateConflict},

			expectErr: core.ErrRecapConflict,
		},
		{
			name: "Error/Dao",

			request: request(sources),

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[struct{}]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request(sources),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/ProjectBytes",

			request: request(sources),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
			bytesMock: &daoMock[*dao.ProjectBytes]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(sources),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockBeatRecapWriteDaoLock(t)
			mockDao := coremocks.NewMockBeatRecapWriteDao(t)
			mockBytes := coremocks.NewMockBeatRecapWriteDaoBytes(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatRecapUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == beatID1 && string(req.Recap) == string(recapOutput) &&
							req.BeatUpdatedAt.Equal(updatedAt) && string(req.Sources) == string(storedSources)
					})).
					Return(testCase.daoMock.err)
			}

			if testCase.bytesMock != nil {
				mockBytes.EXPECT().
					Exec(mock.Anything, &dao.ProjectBytesRequest{ID: projectID}).
					Return(testCase.bytesMock.resp, testCase.bytesMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewBeatRecapWrite(mockLock, mockDao, mockBytes, transactor)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockLock.AssertExpectations(t)
			mockDao.AssertExpectations(t)
			mockBytes.AssertExpectations(t)
		})
	}
}
