package lib_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
)

var (
	genAIGenerationID = uuid.MustParse("00000000-0000-0000-0000-000000000601")
	genAIOwnerID      = uuid.MustParse("00000000-0000-0000-0000-000000000042")
	genAICreatedAt    = time.Date(2026, 7, 28, 12, 0, 0, 123456000, time.UTC)
)

func genAIWireGeneration() *servicegenai.Generation {
	return &servicegenai.Generation{
		Id: genAIGenerationID.String(), OwnerId: genAIOwnerID.String(),
		Purpose: "studio.generation", Status: servicegenai.GenerationStatusPending,
		Attempt: 1, MaxAttempts: 2,
		CreatedAt: genAICreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: genAICreatedAt.Add(time.Second).Format(time.RFC3339Nano),
	}
}

func assertGenAIContext(t *testing.T, parent, forwarded context.Context) {
	t.Helper()

	wantDeadline, _ := parent.Deadline()
	gotDeadline, ok := forwarded.Deadline()
	require.True(t, ok)
	require.Equal(t, wantDeadline, gotDeadline)

	wantMetadata, _ := metadata.FromOutgoingContext(parent)
	gotMetadata, _ := metadata.FromOutgoingContext(forwarded)
	require.Equal(t, wantMetadata, gotMetadata)
}

type genAIWatchStream struct {
	grpc.ClientStream

	receive func() (*servicegenai.GenerationWatchResponse, error)
}

func (stream *genAIWatchStream) Recv() (*servicegenai.GenerationWatchResponse, error) {
	return stream.receive()
}
