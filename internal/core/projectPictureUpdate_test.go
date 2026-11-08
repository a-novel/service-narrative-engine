package core_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/picture"
	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestProjectPictureUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	logo := newJPEG(64, 64)

	request := func(name string, data []byte) *core.ProjectPictureUpdateRequest {
		return &core.ProjectPictureUpdateRequest{OwnerID: ownerID, ProjectID: projectID, Picture: name, Data: data}
	}

	testCases := []struct {
		name string

		request *core.ProjectPictureUpdateRequest

		lockMock *daoMock[bool]
		// mediaType is the media type the upsert stores.
		mediaType  string
		upsertMock *daoMock[*dao.Project]
		transactor transaction.Transactor

		expect    *core.Project
		expectErr error
	}{
		{
			name: "Success/Logo",

			request: request(core.ProjectPictureLogo, logo),

			lockMock:   &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Success/LogoWidest",

			request: request(core.ProjectPictureLogo, newJPEG(512, 512)),

			lockMock:   &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Success/BackgroundWidest",

			request: request(core.ProjectPictureBackground, newJPEG(1280, 720)),

			lockMock:   &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{resp: daoProject()},

			expect: coreProject(),
		},
		{
			name: "Error/LogoRatio",

			request: request(core.ProjectPictureLogo, newJPEG(64, 32)),

			expectErr: picture.ErrRatio,
		},
		{
			name: "Error/LogoDimensions",

			request: request(core.ProjectPictureLogo, newJPEG(513, 513)),

			expectErr: picture.ErrDimensions,
		},
		{
			name: "Error/BackgroundRatio",

			request: request(core.ProjectPictureBackground, logo),

			expectErr: picture.ErrRatio,
		},
		{
			name: "Error/BackgroundDimensions",

			request: request(core.ProjectPictureBackground, newJPEG(1296, 729)),

			expectErr: picture.ErrDimensions,
		},
		{
			name: "Error/Format",

			request: request(core.ProjectPictureLogo, []byte("not an image")),

			expectErr: picture.ErrFormat,
		},
		{
			// A background may hold more bytes than a logo, so the check reaches the decoder.
			name: "Error/BackgroundOverLogoBytes",

			request: request(core.ProjectPictureBackground, make([]byte, core.ProjectLogoBytesMax+1)),

			expectErr: picture.ErrFormat,
		},
		{
			name: "Error/LogoBytes",

			request: request(core.ProjectPictureLogo, make([]byte, core.ProjectLogoBytesMax+1)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BackgroundBytes",

			request: request(core.ProjectPictureBackground, make([]byte, core.ProjectBackgroundBytesMax+1)),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownPicture",

			request: request("banner", logo),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoData",

			request: request(core.ProjectPictureLogo, nil),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request(core.ProjectPictureLogo, logo),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Lock",

			request: request(core.ProjectPictureLogo, logo),

			lockMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Upsert",

			request: request(core.ProjectPictureLogo, logo),

			lockMock:   &daoMock[bool]{resp: true},
			mediaType:  "image/jpeg",
			upsertMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Transaction",

			request: request(core.ProjectPictureLogo, logo),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectPictureUpdateDaoLock(t)
			mockUpsert := coremocks.NewMockProjectPictureUpdateDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.upsertMock != nil {
				mockUpsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectPictureUpsertRequest) bool {
						return req.ProjectID == projectID && req.Picture == testCase.request.Picture &&
							bytes.Equal(req.Data, testCase.request.Data) && req.MediaType == testCase.mediaType &&
							recent(req.Now)
					})).
					Return(testCase.upsertMock.resp, testCase.upsertMock.err)
			}

			transactor := testCase.transactor
			if transactor == nil {
				transactor = transactiontest.NewTransactor()
			}

			service := core.NewProjectPictureUpdate(mockLock, mockUpsert, transactor)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockLock.AssertExpectations(t)
			mockUpsert.AssertExpectations(t)
		})
	}
}
