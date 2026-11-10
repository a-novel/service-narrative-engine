package core_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectLogoUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	logo := newJPEG(64, 64)

	request := func(data []byte) *core.ProjectLogoUpdateRequest {
		return &core.ProjectLogoUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Data: data}
	}

	testCases := []struct {
		name string

		request *core.ProjectLogoUpdateRequest

		ownerMock *daoMock[bool]
		// mediaType is the media type the upsert stores.
		mediaType  string
		upsertMock *daoMock[*dao.Project]

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success/Logo",

			request: request(logo),

			ownerMock:  &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Success/PNG",

			request: request(newPNG(64, 64)),

			ownerMock:  &daoMock[bool]{resp: true},
			mediaType:  "image/png",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Success/LogoWidest",

			request: request(newJPEG(1024, 128)),

			ownerMock:  &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Success/LogoAnyRatio",

			request: request(newJPEG(32, 128)),

			ownerMock:  &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Success/LogoAnyWidth",

			request: request(newJPEG(4096, 64)),

			ownerMock:  &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Error/LogoHeight",

			request: request(newJPEG(64, 129)),

			expectErr: picture.ErrDimensions,
		},
		{
			name: "Error/Format",

			request: request([]byte("not an image")),

			expectErr: picture.ErrFormat,
		},
		{
			name: "Error/LogoBytes",

			request: request(make([]byte, 256<<10+1)),

			expectErr: picture.ErrSize,
		},
		{
			name: "Error/NoData",

			request: request(nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(logo),

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Owner",

			request: request(logo),

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Upsert",

			request: request(logo),

			ownerMock:  &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockProjectLogoUpdateDaoOwner(t)
			mockUpsert := coremocks.NewMockProjectLogoUpdateDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.upsertMock != nil {
				mockUpsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectLogoUpsertRequest) bool {
						return req.ProjectID == projectID && bytes.Equal(req.Data, testCase.request.Data) &&
							req.MediaType == testCase.mediaType && recent(req.Now)
					})).
					Return(testCase.upsertMock.resp, testCase.upsertMock.err)
			}

			service := core.NewProjectLogoUpdate(mockOwner, mockUpsert)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockUpsert.AssertExpectations(t)
		})
	}
}
