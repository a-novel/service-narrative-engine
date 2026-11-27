package core_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// TestInstanceContentUpdate also covers the document check every write of a document runs: each rule of it has
// its case here, on the content.
func TestInstanceContentUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	request := &core.InstanceContentUpdateRequest{
		OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Version: "1.2.0",
		Content: json.RawMessage(`{"beats":[]}`), UpdatedAt: documentUpdatedAt,
	}

	withContent := func(content string) *core.InstanceContentUpdateRequest {
		changed := *request
		changed.Content = json.RawMessage(content)

		return &changed
	}

	withVersion := func(version string) *core.InstanceContentUpdateRequest {
		changed := *request
		changed.Version = version

		return &changed
	}

	// atCap is a document of exactly the most bytes the tests allow.
	atCap := `"` + strings.Repeat("a", caps.ContentBytes-2) + `"`

	// longestVersion is a version of exactly the most characters an instance pins.
	longestVersion := "1.2.0-" + strings.Repeat("a", core.InstanceVersionMax-len("1.2.0-"))

	testCases := []struct {
		name string

		request *core.InstanceContentUpdateRequest

		lockMock   *daoMock[bool]
		daoMock    *daoMock[*dao.InstanceDescriptor]
		bytesMock  *daoMock[*dao.ProjectBytes]
		transactor transaction.Transactor

		expect      *core.InstanceDescriptor
		expectErr   error
		expectField *core.FieldError
	}{
		{
			name: "Success",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Success/Prerelease",

			request: withVersion("2.0.0-dev.1"),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Success/LongestVersion",

			request: withVersion(longestVersion),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Success/MostBytes",

			request: withContent(atCap),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			// Brackets inside a string open nothing, so the document nests three levels and no more.
			name: "Success/DeepestDocument",

			request: withContent(`{"a":[{"b":"[[[{{{"}]}`),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			// An escaped backslash before u0000 spells six characters, and no NUL.
			name: "Success/EscapedBackslash",

			request: withContent(`{"a":"\\u0000","b":"\""}`),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			// Two escapes spell one character past the basic plane: a high half, then its low half.
			name: "Success/SurrogatePair",

			request: withContent(`{"a":"\uD83D\ude00"}`),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			// The database stores 1e60 as 61 digits, which this document of 4 bytes brings to the cap.
			name: "Success/MostBytesStored",

			request: withContent(`1e60`),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Success/Exponents",

			request: withContent(`{"a":[1.5e-7,-2E+3,0.25]}`),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			// The service stores any JSON value, and never reads what it means.
			name: "Success/Scalar",

			request: withContent(`null`),

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: withinBudget,

			expect: coreDescriptor(),
		},
		{
			name: "Error/NoContent",

			request: &core.InstanceContentUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Version: "1.2.0", UpdatedAt: documentUpdatedAt,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Version",

			request: withVersion("1.2"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/VersionTooLong",

			request: withVersion(longestVersion + "a"),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoUpdatedAt",

			request: &core.InstanceContentUpdateRequest{
				OwnerID: ownerID, ProjectID: projectID, ID: instanceID, Version: "1.2.0",
				Content: json.RawMessage(`{}`),
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Content/TooLarge",

			request: withContent(atCap[:len(atCap)-1] + `a"`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxbytes"},
		},
		{
			// A short document the database stores as a long one: it writes a number in full.
			name: "Error/Content/TooLargeStored",

			request: withContent(`1e61`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxbytes"},
		},
		{
			// 20 digits and an exponent of 16363 make the most digits the database stores for a number, so the
			// number passes, and only its size refuses the document.
			name: "Error/Content/MostDigits",

			request: withContent(`[-1234567890.1234567890e16363]`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxbytes"},
		},
		{
			name: "Error/Content/TooManyDigits",

			request: withContent(`[-1234567890.1234567890e16364]`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxdigits"},
		},
		{
			name: "Error/Content/TooManyDecimals",

			request: withContent(`{"a":1E-16383}`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxdigits"},
		},
		{
			// An exponent no integer holds is counted without a fault.
			name: "Error/Content/ExponentPastInteger",

			request: withContent(`1e99999999999999999999999999999999`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxdigits"},
		},
		{
			name: "Error/Content/TooDeep",

			request: withContent(`[[[[]]]]`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "maxdepth"},
		},
		{
			name: "Error/Content/NotJSON",

			request: withContent(`{"beats":`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "json"},
		},
		{
			// The standard library takes any bytes inside a string.
			name: "Error/Content/NotUTF8",

			request: withContent("{\"a\":\"\xff\xfe\"}"),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "json"},
		},
		{
			name: "Error/Content/Nul",

			request: withContent(`{"beats":["a\u0000b"]}`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "nonul"},
		},
		{
			// PostgreSQL refuses the NUL character in a key as it does in a value.
			name: "Error/Content/NulKey",

			request: withContent(`{"\u0000":1}`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "nonul"},
		},
		{
			// A high half of a surrogate pair, and no low half after it. The standard library takes it for JSON,
			// and PostgreSQL refuses to store it.
			name: "Error/Content/LoneHighSurrogate",

			request: withContent(`{"a":"\uD83Da"}`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "nolonesurrogate"},
		},
		{
			// A low half, and no high half before it.
			name: "Error/Content/LoneLowSurrogate",

			request: withContent(`{"a":"\ude00"}`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "nolonesurrogate"},
		},
		{
			// A low half only pairs with the high half right before it.
			name: "Error/Content/SurrogatePairReversed",

			request: withContent(`{"a":"\ude00\ud83d"}`),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "content", Rule: "nolonesurrogate"},
		},
		{
			name: "Error/NotOwned",

			request: request,

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/NotFound",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: dao.ErrInstanceNotFound},

			expectErr: core.ErrInstanceNotFound,
		},
		{
			name: "Error/Stale",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: dao.ErrInstanceContentUpdateStale},

			expectErr: core.ErrInstanceStale,
		},
		{
			name: "Error/Internal",

			request: request,

			lockMock: &daoMock[bool]{resp: true},
			daoMock:  &daoMock[*dao.InstanceDescriptor]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: request,

			lockMock:  &daoMock[bool]{resp: true},
			daoMock:   &daoMock[*dao.InstanceDescriptor]{resp: daoDescriptor()},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/Transaction",

			request: request,

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectLockDao(t)
			mockDao := coremocks.NewMockInstanceContentUpdateDao(t)
			mockBytes := coremocks.NewMockProjectBytesDao(t)

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.InstanceContentUpdateRequest) bool {
						return req.ProjectID == projectID && req.ID == instanceID &&
							req.Version == testCase.request.Version &&
							string(req.Content) == string(testCase.request.Content) &&
							req.UpdatedAt.Equal(documentUpdatedAt) && recent(req.Now)
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
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

			service := core.NewInstanceContentUpdate(mockLock, mockDao, mockBytes, transactor, caps)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			requireFieldError(t, testCase.expectField, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}
