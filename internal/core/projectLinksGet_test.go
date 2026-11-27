package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestProjectLinksGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.ProjectLinksGetRequest{OwnerID: ownerID, ProjectID: projectID}

	const links = `[{"source": {"id": "00000000-0000-0000-0000-000000000031", "target": "a"}}]`

	testCases := []struct {
		name string

		request *core.ProjectLinksGetRequest

		ownerMock *daoMock[bool]
		daoMock   *daoMock[*dao.ProjectLinks]

		expect    *core.ProjectLinks
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock: &daoMock[*dao.ProjectLinks]{resp: &dao.ProjectLinks{
				ProjectID: projectID, Links: json.RawMessage(links), UpdatedAt: documentUpdatedAt,
			}},

			expect: &core.ProjectLinks{Links: json.RawMessage(links), UpdatedAt: new(documentUpdatedAt)},
		},
		{
			// A project that never wrote its links reads as an empty list no write has stamped.
			name: "Success/Unwritten",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.ProjectLinks]{err: dao.ErrProjectLinksNotFound},

			expect: &core.ProjectLinks{Links: json.RawMessage(`[]`)},
		},
		{
			name: "Error/NoProject",

			request: &core.ProjectLinksGetRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Internal",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.ProjectLinks]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectOwnerDao(t)
			mockDao := coremocks.NewMockProjectLinksGetDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.ProjectLinksSelectRequest{ProjectID: projectID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewProjectLinksGet(mockOwner, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
