package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

func TestManuscriptCreate(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errDAO := errors.New("dao failure")
	errTx := errors.New("transaction failure")
	validRequest := &core.ManuscriptCreateRequest{
		Actor:      core.Actor{UserID: ownerID},
		ProjectID:  projectID,
		Manuscript: staticManuscriptValue,
	}

	testCases := []struct {
		name string

		request     *core.ManuscriptCreateRequest
		accessErr   error
		daoErr      error
		transactErr error
		callAccess  bool
		callDAO     bool
		nilEntity   bool

		expectErr error
	}{
		{name: "Success/StaticContractAndFreeformMetadata", request: validRequest, callAccess: true, callDAO: true},
		{
			name: "Success/EmptyPartialManuscript",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: json.RawMessage(`{}`),
			},
			callAccess: true, callDAO: true,
		},
		{
			name: "Success/UnicodeAndExactByteLimit",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: manuscriptOfSize(schemas.ContentDocumentMaxBytes),
			},
			callAccess: true, callDAO: true,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.ManuscriptCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccessBeforeContentValidation",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: json.RawMessage(`{`),
			},
			callAccess: true, accessErr: errAccess, expectErr: errAccess,
		},
		{
			name: "Error/UnknownStaticField",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: json.RawMessage(`{"unexpected":true}`),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/TextBlockOverLimit",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{"text":"` +
						strings.Repeat("界", 32_769) + `","marks":[]}}]}`,
				),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidMarkRange",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: json.RawMessage(
					`{"blocks":[{"type":"text","metadata":{},"data":{"text":"short",` +
						`"marks":[{"type":"bold","start":0,"end":6}]}}]}`,
				),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/OverByteLimit",
			request: &core.ManuscriptCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Manuscript: manuscriptOfSize(schemas.ContentDocumentMaxBytes + 1),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/OwnerChangedBeforeWrite", request: validRequest,
			callAccess: true, callDAO: true, daoErr: dao.ErrProjectLockNotFound,
			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/DAO", request: validRequest,
			callAccess: true, callDAO: true, daoErr: errDAO, expectErr: errDAO,
		},
		{
			name: "Error/Transaction", request: validRequest,
			callAccess: true, transactErr: errTx, expectErr: errTx,
		},
		{
			name: "Error/MissingEntity", request: validRequest,
			callAccess: true, callDAO: true, nilEntity: true,
			expectErr: errors.New("manuscript insert returned no entity"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			manuscriptDao := coremocks.NewMockManuscriptInsertDao(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor: testCase.request.Actor, ProjectID: testCase.request.ProjectID,
					}).
					Return(projectFixture(), testCase.accessErr)
			}

			if testCase.callDAO {
				manuscriptDao.EXPECT().
					Exec(mock.Anything, mock.Anything).
					RunAndReturn(func(
						_ context.Context,
						request *dao.ManuscriptInsertRequest,
					) (*dao.Manuscript, error) {
						if testCase.daoErr != nil || testCase.nilEntity {
							return nil, testCase.daoErr
						}

						require.Equal(t, testCase.request.ProjectID, request.ProjectID)
						require.Equal(t, ownerID, request.OwnerID)
						require.JSONEq(t, string(testCase.request.Manuscript), string(request.Value))

						return &dao.Manuscript{
							ID: request.ID, ProjectID: request.ProjectID,
							Value: request.Value, CreatedAt: request.Now,
						}, nil
					})
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactErr)
			}

			result, err := core.NewManuscriptCreate(projectAccess, manuscriptDao, transactor).
				Exec(t.Context(), testCase.request)

			if testCase.expectErr == nil {
				require.NoError(t, err)
				require.Equal(t, projectID, result.ProjectID)
				require.JSONEq(t, string(testCase.request.Manuscript), string(result.Manuscript))
			} else if errors.Is(err, testCase.expectErr) {
				require.ErrorIs(t, err, testCase.expectErr)
			} else {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			}
		})
	}
}

func manuscriptOfSize(size int) json.RawMessage {
	const (
		prefix = `{"blocks":[{"type":"text","metadata":{"note":"`
		suffix = `"},"data":{"text":"界","marks":[]}}]}`
	)

	return json.RawMessage(prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix)
}
