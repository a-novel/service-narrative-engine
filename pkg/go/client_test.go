package servicetemplate_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-template/internal/config/env"
	"github.com/a-novel/service-template/pkg/go"
)

func TestClient(t *testing.T) {
	t.Parallel()

	client, err := servicetemplate.NewClient(env.GrpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer client.Close()

	_, err = client.UnaryEcho(t.Context(), &golibproto.UnaryEchoRequest{})
	require.NoError(t, err)
}
