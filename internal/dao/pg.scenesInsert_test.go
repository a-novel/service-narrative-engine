package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgScenesInsert(t *testing.T) {
	t.Parallel()

	// The first beat holds the first three scenes, the second beat the fourth, and the third beat none.
	scenes := composerFixtures().scenes

	newSceneID1 := uuid.MustParse("00000000-0000-0000-0000-000000000014")
	newSceneID2 := uuid.MustParse("00000000-0000-0000-0000-000000000015")
	newSceneID3 := uuid.MustParse("00000000-0000-0000-0000-000000000018")

	// item returns a new scene for a beat, at position.
	item := func(id, beat uuid.UUID, position int) dao.ScenesInsertItem {
		return dao.ScenesInsertItem{ID: id, BeatID: beat, Position: position, Title: "Le débat"}
	}
	// scene returns the new scene the insert stores for an item.
	scene := func(id, beat uuid.UUID, position int) *dao.Scene {
		return &dao.Scene{
			ID: id, ProjectID: projectID, BeatID: beat, Position: position, Title: "Le débat", UpdatedAt: now,
		}
	}

	testCases := []struct {
		name string

		items []dao.ScenesInsertItem

		expect []*dao.Scene
		// expectScenes is the project's scenes after the insert, in narrative order.
		expectScenes []*dao.Scene
		// full fills the first beat up to its last position before the insert.
		full      bool
		expectErr error
	}{
		{
			name: "Success/RunAtPosition",

			items: []dao.ScenesInsertItem{item(newSceneID1, beatID1, 1), item(newSceneID2, beatID1, 1)},

			expect: []*dao.Scene{scene(newSceneID1, beatID1, 1), scene(newSceneID2, beatID1, 2)},
			expectScenes: []*dao.Scene{
				scenes[0], scene(newSceneID1, beatID1, 1), scene(newSceneID2, beatID1, 2),
				sceneAt(scenes[1], 3), sceneAt(scenes[2], 4), scenes[3],
			},
		},
		{
			name: "Success/AcrossBeats",

			items: []dao.ScenesInsertItem{
				item(newSceneID1, beatID3, 0), item(newSceneID2, beatID2, 0), item(newSceneID3, beatID1, 9),
			},

			expect: []*dao.Scene{
				scene(newSceneID1, beatID3, 0), scene(newSceneID2, beatID2, 0), scene(newSceneID3, beatID1, 3),
			},
			expectScenes: []*dao.Scene{
				scenes[0], scenes[1], scenes[2], scene(newSceneID3, beatID1, 3),
				scene(newSceneID2, beatID2, 0), sceneAt(scenes[3], 1),
				scene(newSceneID1, beatID3, 0),
			},
		},
		{
			name: "Success/PositionsOutOfOrder",

			items: []dao.ScenesInsertItem{item(newSceneID1, beatID1, 2), item(newSceneID2, beatID1, 0)},

			expect: []*dao.Scene{scene(newSceneID1, beatID1, 3), scene(newSceneID2, beatID1, 0)},
			expectScenes: []*dao.Scene{
				scene(newSceneID2, beatID1, 0), sceneAt(scenes[0], 1), sceneAt(scenes[1], 2),
				scene(newSceneID1, beatID1, 3), sceneAt(scenes[2], 4), scenes[3],
			},
		},
		{
			name: "Error/BeatFull",

			items: []dao.ScenesInsertItem{item(newSceneID1, beatID1, 0)},

			full:      true,
			expectErr: dao.ErrScenesInsertFull,
		},
		{
			name: "Error/BeatOfOtherProject",

			items: []dao.ScenesInsertItem{item(newSceneID1, beatID2, 0), item(newSceneID2, otherBeatID, 0)},

			expectScenes: scenes[:4],
			expectErr:    dao.ErrScenesInsertBeatNotFound,
		},
	}

	daoScenesInsert := dao.NewPgScenesInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				if testCase.full {
					data.scenes = append(data.scenes, fillerScenes()...)
				}

				insertFixtures(ctx, t, data)

				result, err := daoScenesInsert.Exec(ctx, &dao.ScenesInsertRequest{
					ProjectID: projectID, Scenes: testCase.items, Now: now,
				})
				require.ErrorIs(t, err, testCase.expectErr)
				require.ElementsMatch(t, testCase.expect, result)

				// A failed batch inserts nothing and moves no scene.
				if testCase.expectScenes != nil {
					require.Equal(t, testCase.expectScenes, listScenes(ctx, t, projectID))
				}
			})
		})
	}
}
