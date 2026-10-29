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

	// row returns the expected row of a scene, or of a beat that holds none when title is empty.
	row := func(beat *dao.Beat, title string, content json.RawMessage) *dao.ManuscriptRow {
		expected := &dao.ManuscriptRow{BeatID: beat.ID, BeatTitle: beat.Title, BeatDescription: beat.Description}
		if title != "" {
			expected.SceneTitle = &title
			expected.Content = content
		}

		return expected
	}

	beats := composerFixtures().beats

	testCases := []struct {
		name string

		request *dao.ManuscriptSelectRequest

		expect    []*dao.ManuscriptRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ManuscriptSelectRequest{ProjectID: projectID, LayerID: manuscriptLayerID},

			expect: []*dao.ManuscriptRow{
				row(beats[0], "La fenêtre", json.RawMessage(`{"blocks": []}`)),
				row(beats[0], "La rue", nil),
				row(beats[0], "La porte", narration),
				row(beats[1], "Le retour", nil),
				row(beats[2], "", nil),
			},
		},
		{
			name: "Success/OtherProject",

			request: &dao.ManuscriptSelectRequest{ProjectID: otherProjectID, LayerID: manuscriptLayerID},

			expect: []*dao.ManuscriptRow{row(beats[3], "Other", nil)},
		},
		{
			name: "Success/NoBeats",

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
				for i, expected := range testCase.expect {
					actual := *result[i]
					content := actual.Content
					actual.Content = expected.Content
					require.Equal(t, *expected, actual)

					if expected.Content == nil {
						require.Nil(t, content)
					} else {
						require.JSONEq(t, string(expected.Content), string(content))
					}
				}
			})
		})
	}
}
