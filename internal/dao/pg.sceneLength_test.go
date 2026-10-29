package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgSceneLength(t *testing.T) {
	t.Parallel()

	// The fixtures' cells of the first scene hold {"blocks": []} and {"text": "Marie"}, 14 and 17 characters.
	// jsonb stores this one as {"text": "Héloïse"}, 19 characters over 21 bytes.
	secondSceneCell := &dao.Cell{
		ProjectID: projectID, SceneID: sceneID2, LayerID: notesLayerID, Content: json.RawMessage(`{"text":"Héloïse"}`),
		UpdatedAt: updatedAt,
	}
	otherCell := &dao.Cell{
		ProjectID: otherProjectID, SceneID: otherSceneID, LayerID: otherLayerID,
		Content: json.RawMessage(`{"blocks": []}`), UpdatedAt: updatedAt,
	}

	testCases := []struct {
		name string

		request *dao.SceneLengthRequest

		expect []*dao.SceneLength
	}{
		{
			name: "Success",

			request: &dao.SceneLengthRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID1, sceneID2}},

			expect: []*dao.SceneLength{{SceneID: sceneID1, Length: 31}, {SceneID: sceneID2, Length: 19}},
		},
		{
			name: "Success/NoCells",

			request: &dao.SceneLengthRequest{ProjectID: projectID, IDs: []uuid.UUID{sceneID3}},
		},
		{
			name: "Success/OtherProject",

			request: &dao.SceneLengthRequest{ProjectID: projectID, IDs: []uuid.UUID{otherSceneID}},
		},
	}

	daoSceneLength := dao.NewPgSceneLength()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.cells = append(data.cells, secondSceneCell, otherCell)

				insertFixtures(ctx, t, data)

				result, err := daoSceneLength.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)
			})
		})
	}
}
