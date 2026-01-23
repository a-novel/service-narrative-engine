package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/services"
	servicesmocks "github.com/a-novel/service-narrative-engine/internal/services/mocks"
)

func TestSchemaListVersions(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	schemaID1 := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	schemaID2 := uuid.MustParse("00000000-0000-0000-0000-000000000005")

	type projectSelectMock struct {
		resp *dao.Project
		err  error
	}

	type schemaListVersionsMock struct {
		resp []*dao.SchemaVersion
		err  error
	}

	testCases := []struct {
		name string

		request *services.SchemaListVersionsRequest

		projectSelectMock      *projectSelectMock
		schemaListVersionsMock *schemaListVersionsMock

		expect    []*services.SchemaVersion
		expectErr error
	}{
		{
			name: "Success",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:    projectID,
					Owner: userID,
				},
			},

			schemaListVersionsMock: &schemaListVersionsMock{
				resp: []*dao.SchemaVersion{
					{
						ID:        schemaID1,
						CreatedAt: baseTime,
					},
				},
			},

			expect: []*services.SchemaVersion{
				{
					ID:        schemaID1,
					CreatedAt: baseTime,
				},
			},
		},
		{
			name: "Success/MultipleVersions",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:    projectID,
					Owner: userID,
				},
			},

			schemaListVersionsMock: &schemaListVersionsMock{
				resp: []*dao.SchemaVersion{
					{
						ID:        schemaID1,
						CreatedAt: baseTime,
					},
					{
						ID:        schemaID2,
						CreatedAt: baseTime.Add(time.Hour),
					},
				},
			},

			expect: []*services.SchemaVersion{
				{
					ID:        schemaID1,
					CreatedAt: baseTime,
				},
				{
					ID:        schemaID2,
					CreatedAt: baseTime.Add(time.Hour),
				},
			},
		},
		{
			name: "Success/WithOffset",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          5,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:    projectID,
					Owner: userID,
				},
			},

			schemaListVersionsMock: &schemaListVersionsMock{
				resp: []*dao.SchemaVersion{
					{
						ID:        schemaID1,
						CreatedAt: baseTime,
					},
				},
			},

			expect: []*services.SchemaVersion{
				{
					ID:        schemaID1,
					CreatedAt: baseTime,
				},
			},
		},
		{
			name: "Success/EmptyResult",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:    projectID,
					Owner: userID,
				},
			},

			schemaListVersionsMock: &schemaListVersionsMock{
				resp: []*dao.SchemaVersion{},
			},

			expect: []*services.SchemaVersion{},
		},
		{
			name: "Error/InvalidRequest/MissingProjectID",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       uuid.Nil,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingUserID",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          uuid.Nil,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingModuleID",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingModuleNamespace",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "",
				Limit:           10,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/ModuleIDTooLong",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        string(make([]byte, 129)),
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/ModuleNamespaceTooLong",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: string(make([]byte, 129)),
				Limit:           10,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/MissingLimit",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           0,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/LimitTooLarge",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           129,
				Offset:          0,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetNegative",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          -1,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidRequest/OffsetTooLarge",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          8193,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectNotFound",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				err: dao.ErrProjectSelectNotFound,
			},

			expectErr: dao.ErrProjectSelectNotFound,
		},
		{
			name: "Error/ProjectSelectRepositoryError",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/UserDoesNotOwnProject",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:    projectID,
					Owner: otherUserID,
				},
			},

			expectErr: services.ErrUserDoesNotOwnProject,
		},
		{
			name: "Error/SchemaListVersionsRepositoryError",

			request: &services.SchemaListVersionsRequest{
				ProjectID:       projectID,
				UserID:          userID,
				ModuleID:        "test-module",
				ModuleNamespace: "test-namespace",
				Limit:           10,
				Offset:          0,
			},

			projectSelectMock: &projectSelectMock{
				resp: &dao.Project{
					ID:    projectID,
					Owner: userID,
				},
			},

			schemaListVersionsMock: &schemaListVersionsMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				schemaListVersionsRepository := servicesmocks.NewMockSchemaListVersionsRepository(t)
				projectSelectRepository := servicesmocks.NewMockSchemaListVersionsRepositoryProjectSelect(t)

				if testCase.projectSelectMock != nil {
					projectSelectRepository.EXPECT().
						Exec(mock.Anything, &dao.ProjectSelectRequest{
							ID: testCase.request.ProjectID,
						}).
						Return(testCase.projectSelectMock.resp, testCase.projectSelectMock.err)
				}

				if testCase.schemaListVersionsMock != nil {
					schemaListVersionsRepository.EXPECT().
						Exec(mock.Anything, &dao.SchemaListVersionsRequest{
							ProjectID:       testCase.request.ProjectID,
							ModuleID:        testCase.request.ModuleID,
							ModuleNamespace: testCase.request.ModuleNamespace,
							Limit:           testCase.request.Limit,
							Offset:          testCase.request.Offset,
						}).
						Return(testCase.schemaListVersionsMock.resp, testCase.schemaListVersionsMock.err)
				}

				service := services.NewSchemaListVersions(
					schemaListVersionsRepository,
					projectSelectRepository,
				)

				resp, err := service.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, resp)

				schemaListVersionsRepository.AssertExpectations(t)
				projectSelectRepository.AssertExpectations(t)
			})
		})
	}
}
