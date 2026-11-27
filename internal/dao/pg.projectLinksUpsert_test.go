package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgProjectLinksUpsert(t *testing.T) {
	t.Parallel()

	const nextLinks = `[{"source": {"id": "00000000-0000-0000-0000-000000000032", "target": "characters/b"}, ` +
		`"destination": {"id": "00000000-0000-0000-0000-000000000031", "target": "scenes/a"}, ` +
		`"relationship": "but"}]`

	testCases := []struct {
		name string

		request *dao.ProjectLinksUpsertRequest

		expect *dao.ProjectLinksUpsertRow
		// expectLinks are the links the request's project stores after the write.
		expectLinks *dao.ProjectLinks
		expectErr   error
	}{
		{
			name: "Success",

			request: &dao.ProjectLinksUpsertRequest{
				ProjectID: projectID, Links: json.RawMessage(nextLinks), UpdatedAt: new(documentUpdatedAt), Now: now,
			},

			expect:      &dao.ProjectLinksUpsertRow{UpdatedAt: now},
			expectLinks: &dao.ProjectLinks{ProjectID: projectID, Links: json.RawMessage(nextLinks), UpdatedAt: now},
		},
		{
			// The project of the other owner never wrote its links, so its first write names no instant.
			name: "Success/FirstWrite",

			request: &dao.ProjectLinksUpsertRequest{ProjectID: otherProjectID, Links: json.RawMessage(nextLinks), Now: now},

			expect:      &dao.ProjectLinksUpsertRow{UpdatedAt: now},
			expectLinks: &dao.ProjectLinks{ProjectID: otherProjectID, Links: json.RawMessage(nextLinks), UpdatedAt: now},
		},
		{
			// No client reads an instant from links a project never wrote, so the write has nothing to be stale
			// against: it is the first.
			name: "Success/FirstWrite/NamesAnInstant",

			request: &dao.ProjectLinksUpsertRequest{
				ProjectID: otherProjectID, Links: json.RawMessage(nextLinks), UpdatedAt: new(documentUpdatedAt), Now: now,
			},

			expect:      &dao.ProjectLinksUpsertRow{UpdatedAt: now},
			expectLinks: &dao.ProjectLinks{ProjectID: otherProjectID, Links: json.RawMessage(nextLinks), UpdatedAt: now},
		},
		{
			name: "Error/Stale",

			request: &dao.ProjectLinksUpsertRequest{
				ProjectID: projectID, Links: json.RawMessage(nextLinks), UpdatedAt: new(updatedAt), Now: now,
			},

			expectLinks: instanceFixtures().links[0],
			expectErr:   dao.ErrProjectLinksUpsertStale,
		},
		{
			// The instant compares to the microsecond: the value a JavaScript date keeps names another write.
			name: "Error/Stale/Millisecond",

			request: &dao.ProjectLinksUpsertRequest{
				ProjectID: projectID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(documentUpdatedAt.Truncate(time.Millisecond)), Now: now,
			},

			expectLinks: instanceFixtures().links[0],
			expectErr:   dao.ErrProjectLinksUpsertStale,
		},
		{
			// A write that names no instant expects a project that never wrote its links.
			name: "Error/Stale/Written",

			request: &dao.ProjectLinksUpsertRequest{ProjectID: projectID, Links: json.RawMessage(nextLinks), Now: now},

			expectLinks: instanceFixtures().links[0],
			expectErr:   dao.ErrProjectLinksUpsertStale,
		},
	}

	daoProjectLinksUpsert := dao.NewPgProjectLinksUpsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoProjectLinksUpsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				require.Equal(t, testCase.expectLinks, storedLinks(ctx, t, testCase.request.ProjectID))
				requireStoredBytes(ctx, t)
			})
		})
	}
}
