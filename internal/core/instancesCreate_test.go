package core_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/transaction"
	"github.com/a-novel-kit/golib/transaction/transactiontest"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestInstancesCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	composer := core.InstancesCreateItem{
		Plugin: "scenes-composer", Version: "1.2.0", Name: "Composer", Content: json.RawMessage(`{"beats":[]}`),
	}
	characters := core.InstancesCreateItem{
		Plugin: "characters", Version: "0.1.0-dev.1", Name: "Characters", Content: json.RawMessage(`[]`),
	}

	withItems := func(items ...core.InstancesCreateItem) *core.InstancesCreateRequest {
		return &core.InstancesCreateRequest{OwnerID: ownerID, ProjectID: projectID, Instances: items}
	}

	// changed returns the composer instance once change edited it.
	changed := func(change func(item *core.InstancesCreateItem)) core.InstancesCreateItem {
		item := composer
		change(&item)

		return item
	}

	// longestVersion is a version of exactly the most characters an instance pins.
	longestVersion := "1.2.0-" + strings.Repeat("a", core.InstanceVersionMax-len("1.2.0-"))

	created := []*dao.InstanceDescriptor{daoDescriptor()}

	// holds answers the count of a project that holds this many instances.
	holds := func(count int) *daoMock[*dao.InstanceCount] {
		return &daoMock[*dao.InstanceCount]{resp: &dao.InstanceCount{Count: count}}
	}

	testCases := []struct {
		name string

		request *core.InstancesCreateRequest

		lockMock  *daoMock[bool]
		countMock *daoMock[*dao.InstanceCount]
		daoMock   *daoMock[[]*dao.InstanceDescriptor]
		// defaultMock answers the write that makes the first instance of the request the default.
		defaultMock *daoMock[*dao.Project]
		bytesMock   *daoMock[*dao.ProjectBytes]
		transactor  transaction.Transactor

		expect      []*core.InstanceDescriptor
		expectErr   error
		expectField *core.FieldError
	}{
		{
			// The project holds an instance, so it keeps its default.
			name: "Success",

			request: withItems(composer),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			bytesMock: withinBudget,

			expect: []*core.InstanceDescriptor{coreDescriptor()},
		},
		{
			// The first instance of a project becomes its default.
			name: "Success/FirstInstance",

			request: withItems(composer, characters),

			lockMock:    &daoMock[bool]{resp: true},
			countMock:   holds(0),
			daoMock:     &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			defaultMock: &daoMock[*dao.Project]{resp: daoProject()},
			bytesMock:   withinBudget,

			expect: []*core.InstanceDescriptor{coreDescriptor()},
		},
		{
			// A batch may fill a project.
			name: "Success/MostInstances",

			request: withItems(composer, characters),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			bytesMock: withinBudget,

			expect: []*core.InstanceDescriptor{coreDescriptor()},
		},
		{
			name: "Success/LongestName",

			request: withItems(changed(func(item *core.InstancesCreateItem) {
				item.Name = strings.Repeat("é", core.InstanceNameMax)
			})),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			bytesMock: withinBudget,

			expect: []*core.InstanceDescriptor{coreDescriptor()},
		},
		{
			name: "Success/LongestVersion",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Version = longestVersion })),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			bytesMock: withinBudget,

			expect: []*core.InstanceDescriptor{coreDescriptor()},
		},
		{
			// A name is optional: the client shows one of its own for an instance without.
			name: "Success/NoName",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Name = "" })),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			bytesMock: withinBudget,

			expect: []*core.InstanceDescriptor{coreDescriptor()},
		},
		{
			name: "Error/NoInstance",

			request: withItems(),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// The registry names its plugins with DNS labels, which hold no capital.
			name: "Error/Plugin",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Plugin = "Scenes Composer" })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// An instance pins an exact version, which a range is not.
			name: "Error/Version",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Version = "^1.2.0" })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/VersionTooLong",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Version = longestVersion + "a" })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// A name of only whitespace would show as none, where an empty one makes the client show its own.
			name: "Error/BlankName",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Name = " " })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NameTooLong",

			request: withItems(changed(func(item *core.InstancesCreateItem) {
				item.Name = strings.Repeat("é", core.InstanceNameMax+1)
			})),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoContent",

			request: withItems(changed(func(item *core.InstancesCreateItem) { item.Content = nil })),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// A batch larger than a project can hold fits none.
			name: "Error/TooManyInstances",

			request: withItems(composer, characters, composer, characters),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "instances", Rule: "max"},
		},
		{
			// The second instance breaks the document check, which names it.
			name: "Error/Content",

			request: withItems(composer, changed(func(item *core.InstancesCreateItem) {
				item.Content = json.RawMessage(`[[[[]]]]`)
			})),

			expectErr:   core.ErrInvalidRequest,
			expectField: &core.FieldError{Path: "instances[1].content", Rule: "maxdepth"},
		},
		{
			name: "Error/NotOwned",

			request: withItems(composer, characters),

			lockMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			// The project holds two instances of the three it may, so two more insert nothing.
			name: "Error/Full",

			request: withItems(composer, characters),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(2),

			expectErr: core.ErrInstancesFull,
		},
		{
			name: "Error/Count",

			request: withItems(composer, characters),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: &daoMock[*dao.InstanceCount]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Internal",

			request: withItems(composer, characters),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Default",

			request: withItems(composer, characters),

			lockMock:    &daoMock[bool]{resp: true},
			countMock:   holds(0),
			daoMock:     &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			defaultMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ProjectBudget",

			request: withItems(composer, characters),

			lockMock:  &daoMock[bool]{resp: true},
			countMock: holds(1),
			daoMock:   &daoMock[[]*dao.InstanceDescriptor]{resp: created},
			bytesMock: overBudget,

			expectErr: core.ErrProjectBudget,
		},
		{
			name: "Error/Transaction",

			request: withItems(composer, characters),

			transactor: transactiontest.NewFailingTransactor(errFoo),

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockLock := coremocks.NewMockProjectLockDao(t)
			mockCount := coremocks.NewMockInstancesCreateDaoCount(t)
			mockDao := coremocks.NewMockInstancesCreateDaoInsert(t)
			mockDefault := coremocks.NewMockInstancesCreateDaoDefault(t)
			mockBytes := coremocks.NewMockProjectBytesDao(t)

			// firstID is the ID the service gave the first instance of the request.
			var firstID uuid.UUID

			if testCase.lockMock != nil {
				mockLock.EXPECT().
					Exec(mock.Anything, lockRequest).
					Return(testCase.lockMock.resp, testCase.lockMock.err)
			}

			if testCase.countMock != nil {
				mockCount.EXPECT().
					Exec(mock.Anything, &dao.InstanceCountRequest{ProjectID: projectID}).
					Return(testCase.countMock.resp, testCase.countMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.InstancesInsertRequest) bool {
						matches := req.ProjectID == projectID && recent(req.Now) &&
							insertsInstances(t, testCase.request.Instances, req.Instances)
						if matches {
							firstID = req.Instances[0].ID
						}

						return matches
					})).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.defaultMock != nil {
				mockDefault.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(req *dao.ProjectDefaultUpdateRequest) bool {
						return req.ProjectID == projectID && req.InstanceID == firstID && recent(req.Now)
					})).
					Return(testCase.defaultMock.resp, testCase.defaultMock.err)
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

			service := core.NewInstancesCreate(mockLock, mockCount, mockDao, mockDefault, mockBytes, transactor, caps)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			requireFieldError(t, testCase.expectField, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}

// insertsInstances reports whether the insert carries each instance of the request in its order, under an ID of
// its own. It runs inside a mock matcher, so it reports a mismatch with t.Error.
func insertsInstances(t *testing.T, expect []core.InstancesCreateItem, items []dao.InstancesInsertItem) bool {
	t.Helper()

	if len(items) != len(expect) {
		t.Errorf("insert of %d instances, expected %d", len(items), len(expect))

		return false
	}

	ids := map[uuid.UUID]bool{}

	for index, item := range items {
		ids[item.ID] = true

		same := item.ID.Version() == 7 && item.Plugin == expect[index].Plugin &&
			item.Version == expect[index].Version && item.Name == expect[index].Name &&
			string(item.Content) == string(expect[index].Content)
		if !same {
			t.Errorf("insert of instance %d: %+v, expected %+v", index, item, expect[index])

			return false
		}
	}

	if len(ids) != len(items) {
		t.Errorf("insert of %d instances under %d IDs", len(items), len(ids))
	}

	return len(ids) == len(items)
}
