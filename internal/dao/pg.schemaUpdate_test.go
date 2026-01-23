package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestSchemaUpdate(t *testing.T) {
	initialData := map[string]any{
		"title": "Original Story",
		"content": map[string]any{
			"chapter1": "Once upon a time...",
		},
	}

	updatedData := map[string]any{
		"title": "Updated Story",
		"content": map[string]any{
			"chapter1": "In a land far away...",
			"chapter2": "The adventure continues...",
		},
	}

	ownerID := uuid.MustParse("00000000-0000-0000-0000-000000001000")

	testCases := []struct {
		name string

		fixtures []*dao.Schema

		request *dao.SchemaUpdateRequest

		expect    *dao.Schema
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             initialData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Data: updatedData,
				Now:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Schema{
				ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "1.0.0",
				ModulePreversion: "",
				Source:           dao.SchemaSourceUser,
				Data:             updatedData,
				CreatedAt:        time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/UpdateDataOnly",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             initialData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Data: updatedData,
				Now:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Schema{
				ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "1.0.0",
				ModulePreversion: "",
				Source:           dao.SchemaSourceUser,
				Data:             updatedData,
				CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/UpdateToEmptyData",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             initialData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Data: map[string]any{},
				Now:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Schema{
				ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "1.0.0",
				ModulePreversion: "",
				Source:           dao.SchemaSourceUser,
				Data:             map[string]any{},
				CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Success/UpdateToNilData",

			fixtures: []*dao.Schema{
				{
					ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
					Owner:            &ownerID,
					ModuleID:         "test-module",
					ModuleNamespace:  "test-namespace",
					ModuleVersion:    "1.0.0",
					ModulePreversion: "",
					Source:           dao.SchemaSourceUser,
					Data:             initialData,
					CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.SchemaUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Data: nil,
				Now:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Schema{
				ID:               uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				ProjectID:        uuid.MustParse("00000000-0000-0000-0000-000000000100"),
				Owner:            &ownerID,
				ModuleID:         "test-module",
				ModuleNamespace:  "test-namespace",
				ModuleVersion:    "1.0.0",
				ModulePreversion: "",
				Source:           dao.SchemaSourceUser,
				Data:             map[string]any(nil),
				CreatedAt:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.SchemaUpdateRequest{
				ID:   uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Data: updatedData,
				Now:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrSchemaUpdateNotFound,
		},
	}

	repository := dao.NewSchemaUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				schema, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, schema)
			})
		})
	}
}
