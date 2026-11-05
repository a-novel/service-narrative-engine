package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

const (
	// concurrencyCap is the message genai refuses a call with once the user runs as many as their caps allow.
	concurrencyCap = `usage cap reached: purpose "studio.generation" concurrency: 2 generations already in flight`
	// windowCap is the message genai refuses a call with once the user consumed what a window allows.
	windowCap = `usage cap reached: purpose "studio.generation" over 5h0m0s`
)

// userCaps are the caps genai receives for a user of the default caps.
var userCaps = &servicegenai.GenerationCaps{
	Concurrency: lo.ToPtr[int32](2),
	Windows: []*servicegenai.GenerationCapWindow{
		{
			Duration:     durationpb.New(5 * time.Hour),
			InputTokens:  lo.ToPtr[int64](2_000_000),
			OutputTokens: lo.ToPtr[int64](200_000),
		},
		{
			Duration:     durationpb.New(720 * time.Hour),
			InputTokens:  lo.ToPtr[int64](50_000_000),
			OutputTokens: lo.ToPtr[int64](5_000_000),
		},
	},
}

// userRoles are the roles of a user under the default permissions.
var userRoles = []string{"auth:user"}

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

// genaiAnswer is how genai answers one submit, of the call at a position of the plan, at a variant.
type genaiAnswer struct {
	call    int
	variant uint32

	created bool
	resp    *servicegenai.Generation
	err     error
}

func TestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	targets := []*core.GenerationTarget{
		{SceneID: sceneID1, Field: core.SceneFieldTitle, Mode: core.TargetModeWrite},
	}
	snapshot := &core.ProjectSnapshot{Project: &core.Project{ID: projectID, OwnerID: ownerID, Language: "fr"}}

	running := genaiGeneration(servicegenai.GenerationStatusRunning, "")
	succeeded := genaiGeneration(
		servicegenai.GenerationStatusSucceeded, openaiResponse(core.GenerationKindSuggest, outputText(suggestOutput)),
	)

	// genai holds these calls as succeeded, though their output reads as failed.
	incomplete := genaiGeneration(
		servicegenai.GenerationStatusSucceeded,
		`{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`,
	)
	refused := genaiGeneration(
		servicegenai.GenerationStatusSucceeded,
		openaiResponse(core.GenerationKindSuggest,
			map[string]string{"type": "refusal", "refusal": "I cannot help with that."}),
	)
	malformed := genaiGeneration(servicegenai.GenerationStatusSucceeded, "not json")

	// A window's refusal says when the window frees usage.
	windowRefusal := lo.Must(status.New(codes.ResourceExhausted, windowCap).WithDetails(
		&errdetails.RetryInfo{RetryDelay: durationpb.New(90 * time.Second)},
	)).Err()

	testCases := []struct {
		name string

		request *core.GenerationSubmitRequest

		snapshotMock *daoMock[*core.ProjectSnapshot]
		planMock     *daoMock[*core.ContextPlan]
		// genaiMocks answers each submit genai receives, in order, and caps are the caps each carries.
		genaiMocks []*genaiAnswer
		caps       *servicegenai.GenerationCaps

		expect    *core.Generation
		expectErr error
		// expectQuota is the refusal a generation genai admits no call of returns.
		expectQuota *core.GenerationQuotaError
	}{
		{
			name: "Success",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
				Instruction: "Plus court.",
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{{created: true, resp: running}},
			caps:       userCaps,

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
				Context: submitCall("task 1"),
			}}},
		},
		{
			// An admin's empty caps limit nothing.
			name: "Success/Uncapped",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: []string{"auth:admin"}, Kind: core.GenerationKindSuggest,
				Targets: targets, Variant: 2, Stream: true,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{{variant: 2, created: true, resp: running}},
			caps:       &servicegenai.GenerationCaps{},

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
				Context: submitCall("task 1"),
			}}},
		},
		{
			// A resent request finds the call genai already ran, and its output with it.
			name: "Success/Replay",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{{resp: succeeded}},
			caps:       userCaps,

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result: &core.GenerationResult{
					ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
					Model: openaiModel,
				},
				Context: submitCall("task 1"),
			}}},
		},
		{
			// A resent request replays the call whose output held, and reruns the one whose output was cut
			// short at the next variant.
			name: "Success/RerunRejectedOutput",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1"), submitCall("task 2")},
			}},
			genaiMocks: []*genaiAnswer{
				{resp: succeeded},
				{call: 1, resp: incomplete},
				{call: 1, variant: 1, created: true, resp: running},
			},
			caps: userCaps,

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{
				{
					Result: &core.GenerationResult{
						ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
						Model: openaiModel,
					},
					Context: submitCall("task 1"),
				},
				{
					Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
					Context: submitCall("task 2"),
				},
			}},
		},
		{
			// The variants a call already ran replay, so the walk starts from the request's variant and passes
			// each rejected output until genai runs a new one.
			name: "Success/RerunWalksVariants",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
				Variant: 3,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{
				{variant: 3, resp: refused},
				{variant: 4, resp: malformed},
				{variant: 5, created: true, resp: running},
			},
			caps: userCaps,

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{{
				Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
				Context: submitCall("task 1"),
			}}},
		},
		{
			// A cap that refuses the rerun leaves the call waiting, as it would a first submit.
			name: "Success/RerunWaiting",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1"), submitCall("task 2")},
			}},
			genaiMocks: []*genaiAnswer{
				{created: true, resp: running},
				{call: 1, resp: incomplete},
				{call: 1, variant: 1, err: status.Error(codes.ResourceExhausted, concurrencyCap)},
			},
			caps: userCaps,

			expect: &core.Generation{Kind: core.GenerationKindSuggest, Calls: []*core.GenerationCall{
				{
					Result:  &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
					Context: submitCall("task 1"),
				},
				{Waiting: concurrencyCap, Context: submitCall("task 2")},
			}},
		},
		{
			// genai refuses the second call, so it waits, and the third waits without a submit.
			name: "Success/Waiting",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest,
				Calls: []*core.ContextCall{
					submitCall("task 1"), submitCall("task 2"), submitCall("task 3"),
				},
			}},
			genaiMocks: []*genaiAnswer{
				{created: true, resp: running},
				{call: 1, err: status.Error(codes.ResourceExhausted, concurrencyCap)},
			},
			caps: userCaps,

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
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1"), submitCall("task 2")},
			}},
			genaiMocks: []*genaiAnswer{
				{err: status.Error(codes.ResourceExhausted, concurrencyCap)},
			},
			caps: userCaps,

			expectErr:   core.ErrGenerationQuota,
			expectQuota: &core.GenerationQuotaError{Cap: concurrencyCap},
		},
		{
			// A refusal for a window carries the delay until the window frees usage.
			name: "Error/QuotaWindow",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest,
				Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{{err: windowRefusal}},
			caps:       userCaps,

			expectErr:   core.ErrGenerationQuota,
			expectQuota: &core.GenerationQuotaError{Cap: windowCap, RetryAfter: 90 * time.Second},
		},
		{
			// A caller whose roles set no caps submits nothing.
			name: "Error/NoCaps",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: []string{"auth:anon"}, Kind: core.GenerationKindSuggest,
				Targets: targets,
			},

			expectErr: core.ErrNoGenerationCaps,
		},
		{
			// Only a suggest generation takes targets.
			name: "Error/TargetsOnConsolidate",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindConsolidate, Targets: targets,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Snapshot",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{err: core.ErrProjectNotFound},

			expectErr: core.ErrProjectNotFound,
		},
		{
			name: "Error/Plan",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock:     &daoMock[*core.ContextPlan]{err: core.ErrContextTooLarge},

			expectErr: core.ErrContextTooLarge,
		},
		{
			name: "Error/Unavailable",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{{err: status.Error(codes.Unavailable, "planned downtime")}},
			caps:       userCaps,

			expectErr: core.ErrGenaiUnavailable,
		},
		{
			name: "Error/Genai",

			request: &core.GenerationSubmitRequest{
				OwnerID: ownerID, ProjectID: projectID, Roles: userRoles, Kind: core.GenerationKindSuggest, Targets: targets,
			},

			snapshotMock: &daoMock[*core.ProjectSnapshot]{resp: snapshot},
			planMock: &daoMock[*core.ContextPlan]{resp: &core.ContextPlan{
				Kind: core.GenerationKindSuggest, Calls: []*core.ContextCall{submitCall("task 1")},
			}},
			genaiMocks: []*genaiAnswer{{err: errFoo}},
			caps:       userCaps,

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockSnapshot := coremocks.NewMockGenerationSubmitServiceSnapshot(t)
			mockPlan := coremocks.NewMockGenerationSubmitServicePlan(t)
			mockGenai := coremocks.NewMockGenerationSubmitServiceGenai(t)
			gate := core.NewProjectReadGate(1)

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

			for _, answer := range testCase.genaiMocks {
				task := testCase.planMock.resp.Calls[answer.call].Prompt.Task

				mockGenai.EXPECT().
					GenerationSubmit(mock.Anything, mock.MatchedBy(func(request *servicegenai.GenerationSubmitRequest) bool {
						return bytes.Equal(request.GetPayload(), []byte(submitPayload(task))) &&
							request.GetOwnerId() == ownerID.String() && request.GetPurpose() == "studio.generation" &&
							request.GetProvider() == "openai" && request.GetUserTier() == "" &&
							proto.Equal(request.GetCaps(), testCase.caps) &&
							request.GetVariant() == answer.variant && request.GetStream() == testCase.request.Stream
					})).
					// The read and the plan free their slot of the gate before genai answers, which may take long.
					Run(func(ctx context.Context, _ *servicegenai.GenerationSubmitRequest, _ ...grpc.CallOption) {
						waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
						defer cancel()

						require.NoError(t, gate.Run(waitCtx, func(context.Context) error { return nil }))
					}).
					Return(&servicegenai.GenerationSubmitResponse{Generation: answer.resp, Created: answer.created}, answer.err).
					Once()
			}

			caps, err := core.NewGenerationCapsPolicy(config.PermissionsConfigDefault, config.CapsConfigDefault)
			require.NoError(t, err)

			service := core.NewGenerationSubmit(mockSnapshot, mockPlan, mockGenai, gate, caps)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			// A quota refusal names the cap genai refused the first call for.
			if testCase.expectQuota != nil {
				var quota *core.GenerationQuotaError

				require.ErrorAs(t, err, &quota)
				require.Equal(t, testCase.expectQuota, quota)
			}

			mockSnapshot.AssertExpectations(t)
			mockPlan.AssertExpectations(t)
			mockGenai.AssertExpectations(t)
		})
	}
}
