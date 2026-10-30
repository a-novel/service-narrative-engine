package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

// concurrencyCap is the message genai refuses a call with once the user runs as many as their tier allows.
const concurrencyCap = `usage cap reached: purpose "studio.generation" concurrency: 2 generations already in flight`

// submitCall returns a call of a planned generation, whose task names it.
func submitCall(task string) *core.ContextCall {
	return &core.ContextCall{
		Targets: []*core.ContextTarget{
			{Handle: "T1", SceneID: sceneID1, Field: core.SceneFieldTitle, Mode: core.TargetModeWrite},
		},
		Estimate: 1000,
		Prompt: &core.Prompt{
			Model: "gpt-6.1-terra", Reasoning: "medium", MaxOutputTokens: 64000,
			Developer: "You help an author.", Schema: json.RawMessage(`{"type":"object"}`),
			Frame: "<frame>F</frame>", Neighborhood: "<neighborhood/>", Task: task,
		},
	}
}

// submitPayload is the Responses request genai receives for a call of [submitCall].
func submitPayload(task string) string {
	return `{"model":"gpt-6.1-terra","reasoning":{"effort":"medium"},"max_output_tokens":64000,"store":false,` +
		`"prompt_cache_key":"` + projectID.String() + `","prompt_cache_options":{"mode":"explicit"},"input":[` +
		`{"role":"developer","content":[{"type":"input_text","text":"You help an author."}]},` +
		`{"role":"user","content":[{"type":"input_text","text":"<frame>F</frame>",` +
		`"prompt_cache_breakpoint":{"mode":"explicit"}}]},` +
		`{"role":"user","content":[{"type":"input_text","text":"<neighborhood/>"}]},` +
		`{"role":"user","content":[{"type":"input_text","text":"` + task + `"}]}],` +
		`"text":{"format":{"type":"json_schema","name":"suggest","schema":{"type":"object"},"strict":true}}}`
}

func TestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	targets := []*core.GenerationTarget{
		{SceneID: sceneID1, Field: core.SceneFieldTitle, Mode: core.TargetModeWrite},
	}
	snapshot := &core.ProjectSnapshot{Project: &core.Project{ID: projectID, OwnerID: ownerID, Language: "fr"}}

	running := genaiGeneration(servicegenai.GenerationStatusRunning, "")

	testCases := []struct {
		name string

		request *core.GenerationSubmitRequest

		snapshotMock *daoMock[*core.ProjectSnapshot]
		planMock     *daoMock[*core.ContextPlan]
		// genaiMocks answers each call genai receives, in order, and tier is the tier each names.
		genaiMocks []*daoMock[*servicegenai.Generation]
		tier       string

		expect    *core.Generation
		expectErr error
	}{
		{
			name: "Success",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
				Instruction: "Plus court.",
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{{resp: running}},
			tier:       "default",

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
				Context: submitCall("task 1"),
			}}},
		},
		{
			name: "Success/Admin",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
				Admin: true, Variant: 2,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{{resp: running}},
			tier:       "admin",

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
				Context: submitCall("task 1"),
			}}},
		},
		{
			// A resent request finds the call genai already ran, and its output with it.
			name: "Success/Replay",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse(core.GenerationKindSuggest, outputText(suggestOutput)),
			)}},
			tier: "default",

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result: &core.GenerationResult{
					ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
					Model: openaiModel,
				},
				Context: submitCall("task 1"),
			}}},
		},
		{
			// genai refuses the second call, so it waits, and the third waits without a submit.
			name: "Success/Waiting",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest,
				Calls: []*core.ContextCall{
					submitCall("task 1"), submitCall("task 2"), submitCall("task 3"),
				},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{
				{resp: running},
				{err: status.Error(codes.ResourceExhausted, concurrencyCap)},
			},
			tier: "default",

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{
				{
					Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
					Context: submitCall("task 1"),
				},
				{Waiting: concurrencyCap, Context: submitCall("task 2")},
				{Waiting: concurrencyCap, Context: submitCall("task 3")},
			}},
		},
		{
			name: "Error/Quota",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1"), submitCall("task 2")},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{
				{err: status.Error(codes.ResourceExhausted, concurrencyCap)},
			},
			tier: "default",

			expectErr: core.ErrGenerationQuota,
		},
		{
			// Only a suggest generation takes targets.
			name: "Error/TargetsOnConsolidate",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindConsolidate, Targets: targets,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Snapshot",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{err: core.ErrProjectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Plan",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock:     &daoMock[*core.ContextPlan]{err: core.ErrContextTooLarge},

			expectErr: core.ErrContextTooLarge,
		},
		{
			name: "Error/Unavailable",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{{err: status.Error(codes.Unavailable, "planned downtime")}},
			tier:       "default",

			expectErr: core.ErrGenaiUnavailable,
		},
		{
			name: "Error/Genai",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*daoMock[*servicegenai.Generation]{{err: errFoo}},
			tier:       "default",

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSnapshot := coremocks.NewMockGenerationSubmitServiceSnapshot(t)
			mockPlan := coremocks.NewMockGenerationSubmitServicePlan(t)
			mockGenai := coremocks.NewMockGenerationSubmitServiceGenai(t)

			if testCase.snapshotMock != nil {
				mockSnapshot.EXPECT().
					Exec(mock.Anything, &core.ProjectSnapshotReadRequest{OwnerID: ownerID, ProjectID: projectID}).
					Return(testCase.snapshotMock.resp, testCase.snapshotMock.err)
			}

			if testCase.planMock != nil {
				mockPlan.EXPECT().
					Exec(mock.Anything, &core.GenerationPlanRequest{
						Snapshot: snapshot, Kind: core.GenerationKindSuggest, Targets: targets,
						Instruction: testCase.request.Instruction,
					}).
					Return(testCase.planMock.resp, testCase.planMock.err)
			}

			for index, answer := range testCase.genaiMocks {
				task := testCase.planMock.resp.Calls[index].Prompt.Task

				mockGenai.EXPECT().
					GenerationSubmit(mock.Anything, mock.MatchedBy(func(request *servicegenai.GenerationSubmitRequest) bool {
						return bytes.Equal(request.GetPayload(), []byte(submitPayload(task))) &&
							request.GetOwnerId() == ownerID.String() && request.GetPurpose() == "studio.generation" &&
							request.GetProvider() == "openai" && request.GetUserTier() == testCase.tier &&
							request.GetVariant() == testCase.request.Variant
					})).
					Return(&servicegenai.GenerationSubmitResponse{Generation: answer.resp}, answer.err).
					Once()
			}

			service := core.NewGenerationSubmit(mockSnapshot, mockPlan, mockGenai)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			// A quota refusal names the cap genai refused the first call for.
			if errors.Is(err, core.ErrGenerationQuota) {
				var quota *core.GenerationQuotaError

				require.ErrorAs(t, err, &quota)
				require.Equal(t, concurrencyCap, quota.Cap)
			}

			mockSnapshot.AssertExpectations(t)
			mockPlan.AssertExpectations(t)
			mockGenai.AssertExpectations(t)
		})
	}
}
