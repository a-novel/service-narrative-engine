package core_test

import (
	"errors"
	"io"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

// genaiStream stands in for a genai stream: it returns its messages in order, then its end.
type genaiStream struct {
	grpc.ClientStream

	messages []*servicegenai.GenerationStreamResponse
	// end is what the stream returns once its messages run out. Nil returns io.EOF.
	end error
}

func (stream *genaiStream) Recv() (*servicegenai.GenerationStreamResponse, error) {
	if len(stream.messages) == 0 {
		return nil, lo.CoalesceOrEmpty(stream.end, io.EOF)
	}

	message := stream.messages[0]
	stream.messages = stream.messages[1:]

	return message, nil
}

// providerMessage is a genai stream message carrying an OpenAI event.
func providerMessage(sequence int64, event string) *servicegenai.GenerationStreamResponse {
	return &servicegenai.GenerationStreamResponse{SequenceNumber: sequence, Event: []byte(event)}
}

func TestGenerationStream(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	succeeded := &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusSucceeded, Model: openaiModel}
	cancelled := &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusCancelled}
	running := &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning}

	testCases := []struct {
		name string

		request *core.GenerationStreamRequest

		// stream is what genai opens. Nil skips the call.
		stream    *genaiStream
		streamErr error
		// read is what the read returns once the stream ends. Nil skips the call.
		read    *daoMock[*core.GenerationResult]
		readErr error

		expectEvents []*core.GenerationEvent
		// expectErr is returned before the stream opens.
		expectErr error
		// expectStreamErr cuts the stream after its events.
		expectStreamErr error
	}{
		{
			name: "Success",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{messages: []*servicegenai.GenerationStreamResponse{
				providerMessage(0, `{"type":"response.created"}`),
				providerMessage(1, `{"type":"response.output_text.delta","delta":"Il "}`),
				providerMessage(2, `{"type":"response.output_text.delta","delta":"était"}`),
				providerMessage(3, `{"type":"response.completed","response":{}}`),
			}},
			read: &daoMock[*core.GenerationResult]{resp: succeeded},

			expectEvents: []*core.GenerationEvent{
				{Sequence: 1, Delta: "Il "},
				{Sequence: 2, Delta: "était"},
				{Sequence: 3, Result: succeeded},
			},
		},
		{
			name: "Success/Resume",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID, After: lo.ToPtr[int64](2)},

			stream: &genaiStream{messages: []*servicegenai.GenerationStreamResponse{
				providerMessage(3, `{"type":"response.output_text.delta","delta":"."}`),
				providerMessage(4, `{"type":"response.failed","response":{}}`),
			}},
			read: &daoMock[*core.GenerationResult]{resp: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed, Error: "provider failed",
			}},

			expectEvents: []*core.GenerationEvent{
				{Sequence: 3, Delta: "."},
				{Sequence: 4, Result: &core.GenerationResult{
					ID: genaiID, Status: core.GenerationStatusFailed, Error: "provider failed",
				}},
			},
		},
		{
			// A cancelled call's stream ends without a terminal event.
			name: "Success/Cancelled",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{messages: []*servicegenai.GenerationStreamResponse{
				providerMessage(1, `{"type":"response.output_text.delta","delta":"Il "}`),
			}},
			read: &daoMock[*core.GenerationResult]{resp: cancelled},

			expectEvents: []*core.GenerationEvent{
				{Sequence: 1, Delta: "Il "},
				{Sequence: 1, Result: cancelled},
			},
		},
		{
			name: "Success/CancelledBeforeAnyEvent",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{},
			read:   &daoMock[*core.GenerationResult]{resp: cancelled},

			expectEvents: []*core.GenerationEvent{{Result: cancelled}},
		},
		{
			// The read has not settled the call yet: the client resumes, and reads again.
			name: "Success/UnsettledAfterEnd",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{messages: []*servicegenai.GenerationStreamResponse{
				providerMessage(1, `{"type":"response.completed","response":{}}`),
			}},
			read: &daoMock[*core.GenerationResult]{resp: running},
		},
		{
			name: "Error/Ended",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID, After: lo.ToPtr[int64](3)},

			stream: &genaiStream{},
			read:   &daoMock[*core.GenerationResult]{resp: succeeded},

			expectErr: core.ErrStreamEnded,
		},
		{
			name: "Error/NotStreamed",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{end: status.Error(codes.FailedPrecondition, "generation not submitted for streaming")},

			expectErr: core.ErrGenerationNotStreamed,
		},
		{
			name: "Error/NotStreamedYet",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{},
			read:   &daoMock[*core.GenerationResult]{resp: running},

			expectErr: core.ErrGenerationNotStreamed,
		},
		{
			name: "Error/NotFound",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{end: status.Error(codes.NotFound, "generation not found")},

			expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/GenaiUnavailable",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream:    &genaiStream{},
			streamErr: status.Error(codes.Unavailable, "genai is not configured"),

			expectErr: core.ErrGenaiUnavailable,
		},
		{
			name: "Error/ReadAtEnd",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream:  &genaiStream{},
			read:    &daoMock[*core.GenerationResult]{},
			readErr: errFoo,

			expectErr: errFoo,
		},
		{
			// The client resumes a stream genai cut.
			name: "Error/Cut",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{
				messages: []*servicegenai.GenerationStreamResponse{
					providerMessage(1, `{"type":"response.output_text.delta","delta":"Il "}`),
				},
				end: status.Error(codes.Unavailable, "provider dropped the stream"),
			},

			expectEvents:    []*core.GenerationEvent{{Sequence: 1, Delta: "Il "}},
			expectStreamErr: core.ErrGenaiUnavailable,
		},
		{
			name: "Error/MalformedEvent",

			request: &core.GenerationStreamRequest{OwnerID: ownerID, ID: genaiID},

			stream: &genaiStream{messages: []*servicegenai.GenerationStreamResponse{
				providerMessage(1, `not json`),
			}},

			expectStreamErr: errors.New("decode provider event"),
		},
		{
			name: "Error/InvalidRequest",

			request: &core.GenerationStreamRequest{ID: genaiID},

			expectErr: core.ErrInvalidRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockGenai := coremocks.NewMockGenerationStreamServiceGenai(t)
			mockRead := coremocks.NewMockGenerationStreamServiceRead(t)

			if testCase.stream != nil {
				mockGenai.EXPECT().
					GenerationStream(mock.Anything, &servicegenai.GenerationStreamRequest{
						Id: genaiID.String(), OwnerId: ownerID.String(), StartingAfter: testCase.request.After,
					}).
					Return(testCase.stream, testCase.streamErr)
			}

			if testCase.read != nil {
				mockRead.EXPECT().
					Exec(mock.Anything, &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID}).
					Return(testCase.read.resp, testCase.readErr)
			}

			service := core.NewGenerationStream(mockGenai, mockRead)

			events, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			var (
				received  []*core.GenerationEvent
				streamErr error
			)

			if err == nil {
				for event, eventErr := range events {
					if eventErr != nil {
						streamErr = eventErr

						break
					}

					received = append(received, event)
				}
			}

			require.Equal(t, testCase.expectEvents, received)

			if testCase.expectStreamErr == nil {
				require.NoError(t, streamErr)
			} else {
				require.ErrorContains(t, streamErr, testCase.expectStreamErr.Error())
			}

			mockGenai.AssertExpectations(t)
			mockRead.AssertExpectations(t)
		})
	}
}
