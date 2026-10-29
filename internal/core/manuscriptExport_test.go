package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestManuscriptExport(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	blocks := `[{"kind":"narration","text":"La pluie battait les vitres."}]`
	title := func(value string) *string { return &value }

	request := &core.ManuscriptExportRequest{OwnerID: ownerID, ProjectID: projectID, LayerID: layerID1}

	testCases := []struct {
		name string

		request *core.ManuscriptExportRequest

		ownerMock   *daoMock[bool]
		kindMock    *daoMock[[]*dao.LayerKind]
		projectMock *daoMock[*dao.Project]
		daoMock     *daoMock[[]*dao.ManuscriptRow]

		expect    *core.Manuscript
		expectErr error
	}{
		{
			name: "Success",

			request: request,

			ownerMock:   &daoMock[bool]{resp: true},
			kindMock:    &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{{ID: layerID1, Kind: "manuscript", Count: 2}}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			// The first beat holds two scenes, the second one with an empty cell, and the last none.
			daoMock: &daoMock[[]*dao.ManuscriptRow]{resp: []*dao.ManuscriptRow{
				{
					BeatID: beatID1, BeatTitle: "Ouverture", BeatDescription: "Marie attend.", SceneTitle: title("La fenêtre"),
					Content: json.RawMessage(`{"blocks":` + blocks + `}`),
				},
				{BeatID: beatID1, BeatTitle: "Ouverture", BeatDescription: "Marie attend.", SceneTitle: title("Interlude")},
				{
					BeatID: beatID2, BeatTitle: "Dénouement", SceneTitle: title("Le matin"),
					Content: json.RawMessage(`{"blocks":[]}`),
				},
				{BeatID: beatID3, BeatTitle: "Épilogue"},
			}},

			expect: &core.Manuscript{
				Title:    "Les Heures claires",
				Language: "fr",
				Beats: []*core.ManuscriptBeat{
					{Title: "Ouverture", Description: "Marie attend."},
					{Title: "Dénouement"},
					{Title: "Épilogue"},
				},
				Scenes: []*core.ManuscriptScene{
					{Beat: 0, Title: "La fenêtre", Blocks: json.RawMessage(blocks)},
					{Beat: 0, Title: "Interlude", Blocks: json.RawMessage(`[]`)},
					{Beat: 1, Title: "Le matin", Blocks: json.RawMessage(`[]`)},
				},
			},
		},
		{
			name: "Success/NoBeats",

			request: request,

			ownerMock:   &daoMock[bool]{resp: true},
			kindMock:    &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{{ID: layerID1, Kind: "manuscript", Count: 1}}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			daoMock:     &daoMock[[]*dao.ManuscriptRow]{resp: []*dao.ManuscriptRow{}},

			expect: &core.Manuscript{
				Title: "Les Heures claires", Language: "fr", Beats: []*core.ManuscriptBeat{}, Scenes: []*core.ManuscriptScene{},
			},
		},
		{
			name: "Error/MissingLayerID",

			request: &core.ManuscriptExportRequest{OwnerID: ownerID, ProjectID: projectID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotOwned",

			request: request,

			ownerMock: &daoMock[bool]{resp: false},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/CheckOwner",

			request: request,

			ownerMock: &daoMock[bool]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/LayerNotFound",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{}},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/NotesLayer",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{{ID: layerID1, Kind: "notes", Count: 1}}},

			expectErr: core.ErrLayerNotFound,
		},
		{
			name: "Error/ReadKind",

			request: request,

			ownerMock: &daoMock[bool]{resp: true},
			kindMock:  &daoMock[[]*dao.LayerKind]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectProject",

			request: request,

			ownerMock:   &daoMock[bool]{resp: true},
			kindMock:    &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{{ID: layerID1, Kind: "manuscript", Count: 1}}},
			projectMock: &daoMock[*dao.Project]{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectManuscript",

			request: request,

			ownerMock:   &daoMock[bool]{resp: true},
			kindMock:    &daoMock[[]*dao.LayerKind]{resp: []*dao.LayerKind{{ID: layerID1, Kind: "manuscript", Count: 1}}},
			projectMock: &daoMock[*dao.Project]{resp: daoProject()},
			daoMock:     &daoMock[[]*dao.ManuscriptRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockOwner := coremocks.NewMockManuscriptExportDaoOwner(t)
			mockKind := coremocks.NewMockManuscriptExportDaoKind(t)
			mockProject := coremocks.NewMockManuscriptExportDaoProject(t)
			mockDao := coremocks.NewMockManuscriptExportDao(t)

			if testCase.ownerMock != nil {
				mockOwner.EXPECT().
					Exec(mock.Anything, &dao.ProjectOwnedRequest{ID: projectID, OwnerID: ownerID}).
					Return(testCase.ownerMock.resp, testCase.ownerMock.err)
			}

			if testCase.kindMock != nil {
				mockKind.EXPECT().
					Exec(mock.Anything, &dao.LayerKindRequest{ProjectID: projectID, IDs: []uuid.UUID{layerID1}}).
					Return(testCase.kindMock.resp, testCase.kindMock.err)
			}

			if testCase.projectMock != nil {
				mockProject.EXPECT().
					Exec(mock.Anything, &dao.ProjectSelectRequest{ID: projectID}).
					Return(testCase.projectMock.resp, testCase.projectMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.ManuscriptSelectRequest{ProjectID: projectID, LayerID: layerID1}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewManuscriptExport(mockOwner, mockKind, mockProject, mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockOwner.AssertExpectations(t)
			mockKind.AssertExpectations(t)
			mockProject.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
