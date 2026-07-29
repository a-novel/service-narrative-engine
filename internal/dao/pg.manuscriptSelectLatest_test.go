package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgManuscriptSelectLatest(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 29, 1, 2, 3, 123456000, time.UTC)
	updatedAt := now.Add(3 * time.Second)
	latest := &dao.Manuscript{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000701"),
		IdeaID:    fixtureIdeaID,
		Value:     json.RawMessage(`{"format":"novel","scenes":[]}`),
		CreatedAt: now,
		UpdatedAt: &updatedAt,
	}

	testCases := []struct {
		name string

		insert bool

		expect    *dao.Manuscript
		expectErr error
	}{
		{
			name:   "LatestSaveOrUpdate",
			insert: true,
			expect: latest,
		},
		{
			name:      "NotFound",
			expectErr: dao.ErrManuscriptSelectLatestNotFound,
		},
	}

	operation := dao.NewPgManuscriptSelectLatest()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					insertWalkingSkeletonFixtures(t, ctx)

					if testCase.insert {
						db, err := postgres.GetContext(ctx)
						require.NoError(t, err)

						manuscripts := []*dao.Manuscript{
							latest,
							{
								ID:        uuid.MustParse("00000000-0000-0000-0000-000000000702"),
								IdeaID:    fixtureIdeaID,
								Value:     json.RawMessage(`{"format":"screenplay","scenes":[]}`),
								CreatedAt: now.Add(2 * time.Second),
							},
						}

						_, err = db.NewInsert().Model(&manuscripts).Exec(ctx)
						require.NoError(t, err)
					}

					result, err := operation.Exec(ctx, &dao.ManuscriptSelectLatestRequest{
						IdeaID: fixtureIdeaID,
					})
					require.ErrorIs(t, err, testCase.expectErr)

					if result != nil {
						require.JSONEq(t, string(testCase.expect.Value), string(result.Value))
						result.Value = nil
					}

					if testCase.expect != nil {
						expected := *testCase.expect
						expected.Value = nil
						require.Equal(t, &expected, result)
					} else {
						require.Nil(t, result)
					}
				},
			)
		})
	}
}
