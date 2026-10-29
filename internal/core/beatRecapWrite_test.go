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

		ownerMock *daoMock[bool]
		daoMock   *daoMock[struct{}]

		expectErr error
	}{
		{
			name: "Success",

			request: request(sources),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{},
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

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/BeatNotFound",

			request: request(sources),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{err: dao.ErrBeatRecapUpdateNotFound},

			expectErr: core.ErrBeatNotFound,
		},
		{
			name: "Error/Conflict",

			request: request(sources),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{err: dao.ErrBeatRecapUpdateConflict},

			expectErr: core.ErrRecapConflict,
		},
		{
			name: "Error/Dao",

			request: request(sources),

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[struct{}]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockBeatRecapWriteDaoOwner(t)
			mockDao := coremocks.NewMockBeatRecapWriteDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.BeatRecapUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == beatID1 && string(req.Recap) == string(recapOutput) &&
							req.BeatUpdatedAt.Equal(updatedAt) && string(req.Sources) == string(storedSources)
					})).
					Return(testCase.daoMock.err)
			}

			service := core.NewBeatRecapWrite(mockOwner, mockDao)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockOwner.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
