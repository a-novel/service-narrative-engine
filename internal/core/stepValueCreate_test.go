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

func TestStepValueCreate(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errDAO := errors.New("dao failure")
	errTx := errors.New("transaction failure")
	validRequest := &core.StepValueCreateRequest{
		Actor:     core.Actor{UserID: ownerID},
		ProjectID: projectID,
		Key:       "characters",
		Value:     json.RawMessage(`{"shapeRejectedByEngine":true}`),
	}

	testCases := []struct {
		name string

		request     *core.StepValueCreateRequest
		accessErr   error
		daoErr      error
		transactErr error
		callAccess  bool
		callDAO     bool
		nilEntity   bool

		expectErr error
	}{
		{name: "Success/Object", request: validRequest, callAccess: true, callDAO: true},
		{
			name: "Success/Array",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "array",
				Value: json.RawMessage(`[1,"two",false]`),
			},
			callAccess: true, callDAO: true,
		},
		{
			name: "Success/String",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "string",
				Value: json.RawMessage(`"freeform"`),
			},
			callAccess: true, callDAO: true,
		},
		{
			name: "Success/Number",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "number",
				Value: json.RawMessage(`42`),
			},
			callAccess: true, callDAO: true,
		},
		{
			name: "Success/Boolean",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "boolean",
				Value: json.RawMessage(`true`),
			},
			callAccess: true, callDAO: true,
		},
		{
			name: "Success/Null",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "null",
				Value: json.RawMessage(`null`),
			},
			callAccess: true, callDAO: true,
		},
		{
			name: "Success/ExactByteLimit",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "large",
				Value: contentDocumentOfSize(schemas.ContentDocumentMaxBytes),
			},
			callAccess: true, callDAO: true,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.StepValueCreateRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/BlankKey",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "  ",
				Value: json.RawMessage(`{}`),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/KeyOverLimit",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				Key: strings.Repeat("k", 257), Value: json.RawMessage(`{}`),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccessBeforeJSONValidation",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "secret",
				Value: json.RawMessage(`{`),
			},
			callAccess: true, accessErr: errAccess, expectErr: errAccess,
		},
		{
			name: "Error/MalformedJSON",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "broken",
				Value: json.RawMessage(`{`),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/OverByteLimit",
			request: &core.StepValueCreateRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, Key: "large",
				Value: contentDocumentOfSize(schemas.ContentDocumentMaxBytes + 1),
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
			expectErr: errors.New("step value insert returned no entity"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			stepDao := coremocks.NewMockStepValueInsertDao(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor: testCase.request.Actor, ProjectID: testCase.request.ProjectID,
					}).
					Return(projectFixture(), testCase.accessErr)
			}

			if testCase.callDAO {
				stepDao.EXPECT().
					Exec(mock.Anything, mock.Anything).
					RunAndReturn(func(
						_ context.Context,
						request *dao.StepValueInsertRequest,
					) (*dao.StepValue, error) {
						if testCase.daoErr != nil || testCase.nilEntity {
							return nil, testCase.daoErr
						}

						require.Equal(t, testCase.request.ProjectID, request.ProjectID)
						require.Equal(t, ownerID, request.OwnerID)
						require.Equal(t, testCase.request.Key, request.Key)
						require.JSONEq(t, string(testCase.request.Value), string(request.Value))

						return &dao.StepValue{
							ID: request.ID, ProjectID: request.ProjectID, Key: request.Key,
							Value: request.Value, CreatedAt: request.Now,
						}, nil
					})
			}

			transactor := transactiontest.NewTransactor()
			if testCase.transactErr != nil {
				transactor = transactiontest.NewFailingTransactor(testCase.transactErr)
			}

			result, err := core.NewStepValueCreate(projectAccess, stepDao, transactor).
				Exec(t.Context(), testCase.request)

			if testCase.expectErr == nil {
				require.NoError(t, err)
				require.Equal(t, testCase.request.ProjectID, result.ProjectID)
				require.Equal(t, testCase.request.Key, result.Key)
				require.JSONEq(t, string(testCase.request.Value), string(result.Value))
			} else if errors.Is(err, testCase.expectErr) {
				require.ErrorIs(t, err, testCase.expectErr)
			} else {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			}
		})
	}
}
