package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectEditsList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	otherProjectID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	testCases := []struct {
		name string

		request *core.ProjectEditsListRequest

		daoMock *daoMock[[]*dao.ProjectEditsListRow]

		expect    *core.ProjectEdits
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ProjectEditsListRequest{OwnerID: ownerID},

			daoMock: &daoMock[[]*dao.ProjectEditsListRow]{
				resp: []*dao.ProjectEditsListRow{{ID: projectID, EditedAt: editedAt}, {ID: otherProjectID, EditedAt: updatedAt}},
			},

			expect: &core.ProjectEdits{Edits: map[uuid.UUID]time.Time{projectID: editedAt, otherProjectID: updatedAt}},
		},
		{
			name: "Success/NoProject",

			request: &core.ProjectEditsListRequest{OwnerID: ownerID},

			daoMock: &daoMock[[]*dao.ProjectEditsListRow]{},

			expect: &core.ProjectEdits{Edits: map[uuid.UUID]time.Time{}},
		},
		{
			name: "Error/NoOwner",

			request: &core.ProjectEditsListRequest{},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Internal",

			request: &core.ProjectEditsListRequest{OwnerID: ownerID},

			daoMock: &daoMock[[]*dao.ProjectEditsListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockProjectEditsListDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.ProjectEditsListRequest{OwnerID: ownerID, Limit: core.ProjectsPerOwnerMax}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectEditsList(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockDao.AssertExpectations(t)
		})
	}
}
