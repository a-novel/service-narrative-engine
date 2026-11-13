package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectFieldsDrift(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	removedID := uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	// held names the test project at an instant with microseconds, at the widest offset PostgreSQL takes.
	held := map[string]string{projectID.String(): "2021-01-02T01:00:00.123456+15:59"}

	// full holds a project past the most a user owns.
	full := make(map[string]string, core.ProjectsPerOwnerMax+1)
	for range core.ProjectsPerOwnerMax + 1 {
		full[uuid.NewString()] = "2021-01-02T00:00:00Z"
	}

	testCases := []struct {
		name string

		request *core.ProjectFieldsDriftRequest

		daoRequest *dao.ProjectFieldsDriftRequest
		daoMock    *daoMock[[]*dao.ProjectFieldsDriftRow]

		expect    *core.ProjectFieldsDriftResult
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: held},

			daoRequest: &dao.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: held, Limit: 101},
			daoMock: &daoMock[[]*dao.ProjectFieldsDriftRow]{
				resp: []*dao.ProjectFieldsDriftRow{{Project: *daoProject()}, {RemovedID: &removedID}},
			},

			expect: &core.ProjectFieldsDriftResult{Projects: []*core.Project{coreProject()}, Removed: []uuid.UUID{removedID}},
		},
		{
			name: "Success/NoDrift",

			request: &core.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: map[string]string{}},

			daoRequest: &dao.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: map[string]string{}, Limit: 100},
			daoMock:    &daoMock[[]*dao.ProjectFieldsDriftRow]{},

			expect: &core.ProjectFieldsDriftResult{Projects: []*core.Project{}, Removed: []uuid.UUID{}},
		},
		{
			name: "Error/NoOwner",

			request: &core.ProjectFieldsDriftRequest{Projects: held},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoProjects",

			request: &core.ProjectFieldsDriftRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TooManyProjects",

			request: &core.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: full},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidID",

			request: &core.ProjectFieldsDriftRequest{
				OwnerID: ownerID, Projects: map[string]string{"not-a-uuid": "2021-01-02T00:00:00Z"},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidUpdatedAt",

			request: &core.ProjectFieldsDriftRequest{
				OwnerID: ownerID, Projects: map[string]string{projectID.String(): "2021-01-02 00:00:00"},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			// PostgreSQL holds no year 0.
			name: "Error/YearZero",

			request: &core.ProjectFieldsDriftRequest{
				OwnerID: ownerID, Projects: map[string]string{projectID.String(): "0000-01-01T00:00:00Z"},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			// PostgreSQL takes offsets under 16 hours, east or west.
			name: "Error/OffsetEast",

			request: &core.ProjectFieldsDriftRequest{
				OwnerID: ownerID, Projects: map[string]string{projectID.String(): "2026-11-04T08:15:30+16:00"},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/OffsetWest",

			request: &core.ProjectFieldsDriftRequest{
				OwnerID: ownerID, Projects: map[string]string{projectID.String(): "2026-11-04T08:15:30-16:00"},
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Internal",

			request: &core.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: held},

			daoRequest: &dao.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: held, Limit: 101},
			daoMock:    &daoMock[[]*dao.ProjectFieldsDriftRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockProjectFieldsDriftDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectFieldsDrift(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
		})
	}
}
