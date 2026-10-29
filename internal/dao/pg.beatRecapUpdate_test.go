package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgBeatRecapUpdate(t *testing.T) {
	t.Parallel()

	recap := &dao.Recap{
		Condensed: "Marie attend.", Compact: "Marie attend.", Essentials: "Marie.", Names: []string{"Marie"},
	}
	recapJSON, err := json.Marshal(recap)
	require.NoError(t, err)

	// The first beat's scenes, of which only the first holds a manuscript cell.
	sources := []*dao.BeatRecapSource{
		{SceneID: sceneID1, CellUpdatedAt: updatedAt}, {SceneID: sceneID2}, {SceneID: sceneID3},
	}
	sourcesJSON, err := json.Marshal(sources)
	require.NoError(t, err)

	older := &dao.Recap{Condensed: "Avant.", Compact: "Avant.", Essentials: "Avant.", Names: []string{}}
	olderSources := []*dao.BeatRecapSource{{SceneID: sceneID1, CellUpdatedAt: createdAt}}

	// recapped returns a copy of a beat holding a recap that read the beat at stamp.
	recapped := func(beat *dao.Beat, recap *dao.Recap, stamp time.Time, sources []*dao.BeatRecapSource) *dao.Beat {
		copied := *beat
		copied.Recap, copied.RecapBeatUpdatedAt, copied.RecapSources = recap, &stamp, sources

		return &copied
	}

	request := func(id uuid.UUID, stamp time.Time) *dao.BeatRecapUpdateRequest {
		return &dao.BeatRecapUpdateRequest{
			ProjectID: projectID, ID: id, Recap: recapJSON, BeatUpdatedAt: stamp, Sources: sourcesJSON,
		}
	}

	testCases := []struct {
		name string

		// stored marks the first beat as holding an older recap, read at storedAt.
		stored   bool
		storedAt time.Time

		request *dao.BeatRecapUpdateRequest

		// expectBeat is the first beat after the write. The other beats never change.
		expectBeat func(beat *dao.Beat) *dao.Beat
		expectErr  error
	}{
		{
			name: "Success",

			request: request(beatID1, updatedAt),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return recapped(beat, recap, updatedAt, sources) },
		},
		{
			// The beat changed after the consolidation read it: the recap is stored, and stale.
			name: "Success/BeatChangedSince",

			request: request(beatID1, createdAt),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return recapped(beat, recap, createdAt, sources) },
		},
		{
			name: "Success/ReplacesOlder",

			stored: true, storedAt: createdAt,

			request: request(beatID1, updatedAt),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return recapped(beat, recap, updatedAt, sources) },
		},
		{
			name: "Error/OlderThanStored",

			stored: true, storedAt: updatedAt,

			request: request(beatID1, createdAt),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return beat },
			expectErr:  dao.ErrBeatRecapUpdateConflict,
		},
		{
			name: "Error/NewerThanBeat",

			request: request(beatID1, now),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return beat },
			expectErr:  dao.ErrBeatRecapUpdateConflict,
		},
		{
			name: "Error/UnknownBeat",

			request: request(absentID, updatedAt),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return beat },
			expectErr:  dao.ErrBeatRecapUpdateNotFound,
		},
		{
			name: "Error/OtherProjectBeat",

			request: request(otherBeatID, updatedAt),

			expectBeat: func(beat *dao.Beat) *dao.Beat { return beat },
			expectErr:  dao.ErrBeatRecapUpdateNotFound,
		},
	}

	daoBeatRecapUpdate := dao.NewPgBeatRecapUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				if testCase.stored {
					data.beats[0] = recapped(data.beats[0], older, testCase.storedAt, olderSources)
				}

				insertFixtures(ctx, t, data)
				otherBeat := data.beats[3]

				err := daoBeatRecapUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				expect := append([]*dao.Beat{testCase.expectBeat(data.beats[0])}, data.beats[1:3]...)
				require.Equal(t, expect, listBeats(ctx, t, projectID))
				require.Equal(t, []*dao.Beat{otherBeat}, listBeats(ctx, t, otherProjectID))
			})
		})
	}
}
