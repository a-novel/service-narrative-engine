package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgInstanceLinksUpdate(t *testing.T) {
	t.Parallel()

	const nextLinks = `[{"source": "characters/b", ` +
		`"destination": {"id": "00000000-0000-0000-0000-000000000031", "target": "scenes/a"}, ` +
		`"relationship": "but"}]`

	// written returns the links the instances of the test project store once the instance with this ID wrote
	// nextLinks.
	written := func(id uuid.UUID) []*dao.InstanceLinks {
		stored := instanceFixtures().allLinks()

		for index, links := range stored {
			if links.InstanceID == id {
				stored[index] = &dao.InstanceLinks{InstanceID: id, Links: json.RawMessage(nextLinks), UpdatedAt: new(now)}
			}
		}

		return stored
	}

	testCases := []struct {
		name string

		request *dao.InstanceLinksUpdateRequest

		expect *dao.InstanceLinksUpdateRow
		// expectLinks are the links the instances of the test project store after the write.
		expectLinks []*dao.InstanceLinks
		expectErr   error
	}{
		{
			name: "Success",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: instanceID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(linksUpdatedAt), Now: now,
			},

			expect:      &dao.InstanceLinksUpdateRow{UpdatedAt: now},
			expectLinks: written(instanceID),
		},
		{
			// The second instance never wrote its links, so its first write names no instant.
			name: "Success/FirstWrite",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: otherInstanceID, Links: json.RawMessage(nextLinks), Now: now,
			},

			expect:      &dao.InstanceLinksUpdateRow{UpdatedAt: now},
			expectLinks: written(otherInstanceID),
		},
		{
			name: "Error/Stale",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: instanceID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(updatedAt), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceLinksUpdateStale,
		},
		{
			// The instant compares to the microsecond: the value a JavaScript date keeps names another write.
			name: "Error/Stale/Millisecond",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: instanceID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(linksUpdatedAt.Truncate(time.Millisecond)), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceLinksUpdateStale,
		},
		{
			// The links and the content are stamped apart: the stamp of the content names no write of the links.
			name: "Error/Stale/ContentStamp",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: instanceID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(documentUpdatedAt), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceLinksUpdateStale,
		},
		{
			// A write that names no instant expects an instance that never wrote its links.
			name: "Error/Stale/Written",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: instanceID, Links: json.RawMessage(nextLinks), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceLinksUpdateStale,
		},
		{
			// An instance that never wrote its links holds no instant, so a first write that names one is stale.
			name: "Error/Stale/FirstWriteNamesAnInstant",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: otherInstanceID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(linksUpdatedAt), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceLinksUpdateStale,
		},
		{
			name: "Error/NotFound",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: projectID, ID: absentID, Links: json.RawMessage(nextLinks), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.InstanceLinksUpdateRequest{
				ProjectID: otherProjectID, ID: instanceID, Links: json.RawMessage(nextLinks),
				UpdatedAt: new(linksUpdatedAt), Now: now,
			},

			expectLinks: instanceFixtures().allLinks(),
			expectErr:   dao.ErrInstanceNotFound,
		},
	}

	daoInstanceLinksUpdate := dao.NewPgInstanceLinksUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceLinksUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				require.Equal(t, testCase.expectLinks, storedInstanceLinks(ctx, t, projectID))

				// A links write leaves the content of every instance, and the stamp a content write names.
				require.Equal(t, instanceFixtures().instances[:2], projectInstances(ctx, t, projectID))
				requireStoredBytes(ctx, t)
			})
		})
	}
}
