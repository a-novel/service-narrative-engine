package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgManuscriptSelect(t *testing.T) {
	t.Parallel()

	// The fixtures' first scene holds an empty manuscript cell and a notes cell; the third gets text.
	narration := json.RawMessage(`{"blocks": [{"kind": "narration", "text": "La pluie battait les vitres."}]}`)

	testCases := []struct {
		name string

		request *dao.ManuscriptSelectRequest

		expect    []*dao.ManuscriptScene
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ManuscriptSelectRequest{ProjectID: projectID, LayerID: manuscriptLayerID},

			expect: []*dao.ManuscriptScene{
				{Title: "Ouverture", Content: json.RawMessage(`{"blocks": []}`)},
				{Title: "Le thème"},
				{Title: "Le déclencheur", Content: narration},
			},
		},
		{
			name: "Success/OtherProject",

			request: &dao.ManuscriptSelectRequest{ProjectID: otherProjectID, LayerID: manuscriptLayerID},

			expect: []*dao.ManuscriptScene{{Title: "Other"}},
		},
		{
			name: "Success/NoScenes",

			request: &dao.ManuscriptSelectRequest{ProjectID: absentID, LayerID: manuscriptLayerID},
		},
	}

	daoManuscriptSelect := dao.NewPgManuscriptSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.cells = append(data.cells, &dao.Cell{
					ProjectID: projectID, SceneID: sceneID3, LayerID: manuscriptLayerID, Content: narration,
					UpdatedAt: updatedAt,
				})
				insertFixtures(ctx, t, data)

				result, err := daoManuscriptSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Len(t, result, len(testCase.expect))

				// PostgreSQL normalizes the spacing of a jsonb value, so content compares as JSON.
				for i, scene := range testCase.expect {
					require.Equal(t, scene.Title, result[i].Title)

					if scene.Content == nil {
						require.Nil(t, result[i].Content)
					} else {
						require.JSONEq(t, string(scene.Content), string(result[i].Content))
					}
				}
			})
		})
	}
}
