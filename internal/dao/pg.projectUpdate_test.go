package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectUpdate(t *testing.T) {
	t.Parallel()

	update := &dao.ProjectUpdateRequest{
		ID: projectID, Title: "Clear Hours", Premise: "Marie waits.", Language: "en",
		Tags: []string{"thriller", "polar", "noir"}, Genres: []string{"roman noir", "policier"}, Now: now,
	}

	// updated returns the test project after the update, with the version of its logo.
	updated := func(logo *time.Time) *dao.Project {
		return &dao.Project{
			ID: projectID, OwnerID: ownerID, Title: "Clear Hours", Premise: "Marie waits.", Language: "en",
			Tags: []string{"thriller", "polar", "noir"}, Genres: []string{"roman noir", "policier"},
			CreatedAt: createdAt, UpdatedAt: now, EditedAt: updatedAt, LogoUpdatedAt: logo,
		}
	}

	testCases := []struct {
		name string

		fixtures fixtures
		request  *dao.ProjectUpdateRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			fixtures: composerFixtures(),
			request:  update,

			expect: updated(nil),
		},
		{
			name: "Success/Logo",

			fixtures: logoFixtures(),
			request:  update,

			expect: updated(lo.ToPtr(updatedAt)),
		},
		{
			name: "Success/ClearLabels",

			fixtures: composerFixtures(),
			request: &dao.ProjectUpdateRequest{
				ID: projectID, Title: "Clear Hours", Premise: "Marie waits.", Language: "en", Tags: []string{},
				Genres: []string{}, Now: now,
			},

			expect: &dao.Project{
				ID: projectID, OwnerID: ownerID, Title: "Clear Hours", Premise: "Marie waits.", Language: "en",
				Tags: []string{}, Genres: []string{}, CreatedAt: createdAt, UpdatedAt: now, EditedAt: updatedAt,
			},
		},
		{
			name: "Error/NotFound",

			fixtures: logoFixtures(),
			request: &dao.ProjectUpdateRequest{
				ID: absentID, Title: "Clear Hours", Language: "en", Tags: []string{}, Genres: []string{}, Now: now,
			},

			expectErr: dao.ErrProjectUpdateNotFound,
		},
	}

	daoProjectUpdate := dao.NewPgProjectUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				result, err := daoProjectUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				requireStoredBytes(ctx, t)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
