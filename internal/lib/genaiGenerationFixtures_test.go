package lib_test

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

var (
	genAIGenerationID = uuid.MustParse("00000000-0000-0000-0000-000000000601")
	genAIOwnerID      = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	genAICreatedAt    = time.Date(2026, 7, 28, 12, 0, 0, 123456000, time.UTC)
	genAIUpdatedAt    = genAICreatedAt.Add(time.Second)
	genAISettledAt    = genAIUpdatedAt.Add(time.Second)
	genAIExpiresAt    = genAISettledAt.Add(30 * 24 * time.Hour)
)

func genAIWireGeneration(
	status servicegenai.GenerationStatus,
	output []byte,
) *servicegenai.Generation {
	generation := &servicegenai.Generation{
		Id:          genAIGenerationID.String(),
		OwnerId:     genAIOwnerID.String(),
		Purpose:     "studio.generation",
		Status:      status,
		Attempt:     1,
		MaxAttempts: 2,
		Output:      output,
		CreatedAt:   genAICreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   genAIUpdatedAt.Format(time.RFC3339Nano),
	}

	switch status {
	case servicegenai.GenerationStatusSucceeded,
		servicegenai.GenerationStatusFailed,
		servicegenai.GenerationStatusAbandoned,
		servicegenai.GenerationStatusCancelled:
		generation.SettledAt = genAISettledAt.Format(time.RFC3339Nano)
		generation.ExpiresAt = genAIExpiresAt.Format(time.RFC3339Nano)
	}

	return generation
}

func expectedGatewayGeneration(status lib.GenerationStatus, output string) *lib.Generation {
	generation := &lib.Generation{
		ID:          genAIGenerationID,
		OwnerID:     genAIOwnerID,
		Purpose:     "studio.generation",
		Status:      status,
		Attempt:     1,
		MaxAttempts: 2,
		Output:      output,
		CreatedAt:   genAICreatedAt,
		UpdatedAt:   genAIUpdatedAt,
	}

	switch status {
	case lib.GenerationStatusSucceeded,
		lib.GenerationStatusFailed,
		lib.GenerationStatusAbandoned,
		lib.GenerationStatusCancelled:
		generation.SettledAt = &genAISettledAt
		generation.ExpiresAt = &genAIExpiresAt
	}

	return generation
}

func genAIResponsesOutput(t *testing.T, value any) []byte {
	t.Helper()

	text, err := json.Marshal(value)
	require.NoError(t, err)

	output, err := json.Marshal(map[string]any{
		"output": []any{map[string]any{
			"type": "message",
			"content": []any{map[string]any{
				"type": "output_text",
				"text": string(text),
			}},
		}},
	})
	require.NoError(t, err)

	return output
}

func genAIResponsesRefusal(t *testing.T) []byte {
	t.Helper()

	output, err := json.Marshal(map[string]any{
		"output": []any{map[string]any{
			"content": []any{map[string]any{
				"type":    "refusal",
				"refusal": "private provider explanation",
			}},
		}},
	})
	require.NoError(t, err)

	return output
}

type genAIWatchStream struct {
	grpc.ClientStream

	responses []*servicegenai.GenerationWatchResponse
	err       error
	index     int
}

func (stream *genAIWatchStream) Recv() (*servicegenai.GenerationWatchResponse, error) {
	if stream.index < len(stream.responses) {
		response := stream.responses[stream.index]
		stream.index++

		return response, nil
	}

	if stream.err != nil {
		err := stream.err
		stream.err = nil

		return nil, err
	}

	return nil, io.EOF
}
