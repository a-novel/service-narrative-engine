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

func TestPgInstanceLinksList(t *testing.T) {
	t.Parallel()

	// page returns these links of the test project as the rows of a page, each with the total of its list.
	page := func(total int, links ...*dao.InstanceLinks) []*dao.InstanceLinksListRow {
		rows := make([]*dao.InstanceLinksListRow, 0, len(links))

		for _, stored := range links {
			rows = append(rows, &dao.InstanceLinksListRow{InstanceLinks: *stored, Total: total})
		}

		return rows
	}

	stored := instanceFixtures().allLinks()

	testCases := []struct {
		name string

		request *dao.InstanceLinksListRequest

		expect []*dao.InstanceLinksListRow
	}{
		{
			// The first instance wrote its links. The second never did, and comes with an empty list and no instant.
			name: "Success",

			request: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: 10},

			expect: page(2, stored...),
		},
		{
			name: "Success/Limit",

			request: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: 1},

			expect: page(2, stored[0]),
		},
		{
			name: "Success/Offset",

			request: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: 10, Offset: 1},

			expect: page(2, stored[1]),
		},
		{
			// No row carries the total of a page past the end.
			name: "Success/PastTheEnd",

			request: &dao.InstanceLinksListRequest{ProjectID: projectID, Limit: 10, Offset: 2},

			expect: []*dao.InstanceLinksListRow{},
		},
		{
			name: "Success/OtherProject",

			request: &dao.InstanceLinksListRequest{ProjectID: otherProjectID, Limit: 10},

			expect: page(1, &dao.InstanceLinks{InstanceID: alienInstanceID, Links: json.RawMessage(noLinks)}),
		},
		{
			name: "Success/NoProject",

			request: &dao.InstanceLinksListRequest{ProjectID: absentID, Limit: 10},

			expect: []*dao.InstanceLinksListRow{},
		},
	}

	daoInstanceLinksList := dao.NewPgInstanceLinksList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceLinksList.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
