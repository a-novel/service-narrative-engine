package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgInstanceDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		// links are links the second instance wrote before the delete, when the case sets them.
		links   *dao.InstanceLinks
		request *dao.InstanceDeleteRequest

		expect *dao.InstanceDescriptor
		// expectInstances are the instances of the test project after the delete.
		expectInstances []*dao.Instance
		// expectLinks are the links the instances of the test project store after the delete.
		expectLinks []*dao.InstanceLinks
		expectErr   error
	}{
		{
			// The first instance stores a link that points at the deleted one. The link stays where it is stored.
			name: "Success",

			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: otherInstanceID},

			expect:          instanceFixtures().descriptors(otherInstanceID)[0],
			expectInstances: instanceFixtures().instances[:1],
			expectLinks:     instanceFixtures().allLinks()[:1],
		},
		{
			// The links an instance stores go with it.
			name: "Success/StoresLinks",

			links: &dao.InstanceLinks{
				InstanceID: otherInstanceID, Links: json.RawMessage(composerLinks), UpdatedAt: new(linksUpdatedAt),
			},
			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: otherInstanceID},

			expect:          instanceFixtures().descriptors(otherInstanceID)[0],
			expectInstances: instanceFixtures().instances[:1],
			expectLinks:     instanceFixtures().allLinks()[:1],
		},
		{
			// The test project opens on its first instance.
			name: "Error/Default",

			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: instanceID},

			expectInstances: instanceFixtures().instances[:2],
			expectLinks:     instanceFixtures().allLinks(),
			expectErr:       dao.ErrInstanceDeleteDefault,
		},
		{
			name: "Error/NotFound",

			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: absentID},

			expectInstances: instanceFixtures().instances[:2],
			expectLinks:     instanceFixtures().allLinks(),
			expectErr:       dao.ErrInstanceNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.InstanceDeleteRequest{ProjectID: otherProjectID, ID: otherInstanceID},

			expectInstances: instanceFixtures().instances[:2],
			expectLinks:     instanceFixtures().allLinks(),
			expectErr:       dao.ErrInstanceNotFound,
		},
	}

	daoInstanceDelete := dao.NewPgInstanceDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := instanceFixtures()
				if testCase.links != nil {
					data.instanceLinks = append(data.instanceLinks, testCase.links)
				}

				insertFixtures(ctx, t, data)

				result, err := daoInstanceDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				if testCase.expectErr == nil {
					require.Equal(t, testCase.expect, result)
				} else {
					require.Nil(t, result)
				}

				require.Equal(t, testCase.expectInstances, projectInstances(ctx, t, projectID))
				require.Equal(t, testCase.expectLinks, storedInstanceLinks(ctx, t, projectID))
				requireStoredBytes(ctx, t)
			})
		})
	}
}
