package servicetemplate_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/a-novel-kit/golib/grpcf"
	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-template/internal/config/env"
	"github.com/a-novel/service-template/pkg/go"
)

func TestClient(t *testing.T) {
	t.Parallel()

	t.Run("Success/RemoteCalls", func(t *testing.T) {
		t.Parallel()

		client, err := servicetemplate.NewClient(env.GrpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)

		defer client.Close()

		_, err = client.UnaryEcho(t.Context(), &golibproto.UnaryEchoRequest{})
		require.NoError(t, err)
	})

	t.Run("Success/ImportIgnoresServerEnvironment", func(t *testing.T) {
		t.Parallel()

		consumerPath := buildClientConsumer(t)

		testCases := []struct {
			name   string
			prefix string
		}{
			{name: "Unprefixed"},
			{name: "Prefixed", prefix: "TEMPLATE_CLIENT_TEST_"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				// Invalid server settings fail a process that reads them, so the consumer only runs
				// while the client leaves them alone.
				command := exec.CommandContext(t.Context(), consumerPath)

				command.Env = append(os.Environ(),
					"GOWORK=off",
					"SERVICE_TEMPLATE_ENV_PREFIX="+testCase.prefix,
					testCase.prefix+"REST_TIMEOUT_READ=invalid",
					testCase.prefix+"GRPC_TIMEOUT_SHUTDOWN=invalid",
					testCase.prefix+"POSTGRES_PORT=invalid",
					testCase.prefix+"OTEL=invalid",
				)

				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
			})
		}
	})

	t.Run("Success/DependencyBoundary", func(t *testing.T) {
		t.Parallel()

		output, err := exec.CommandContext(t.Context(), "go", "list", "-deps", ".").Output()
		require.NoError(t, err)

		dependencies := strings.Split(strings.TrimSpace(string(output)), "\n")

		const configPath = "github.com/a-novel/service-template/internal/config"

		require.NotContains(t, dependencies, configPath)
		require.NotContains(t, dependencies, configPath+"/env")
	})

	t.Run("Success/PropagatesTraceContext", func(t *testing.T) {
		t.Parallel()

		// golib's otel.Init installs this propagator in every service; the default one sends nothing.
		otel.SetTextMapPropagator(propagation.TraceContext{})

		received := make(chan []string, 1)
		server := grpc.NewServer(grpc.UnaryInterceptor(func(
			ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
		) (any, error) {
			md, _ := metadata.FromIncomingContext(ctx)
			received <- md.Get("traceparent")

			return handler(ctx, req)
		}))
		grpcf.RegisterEchoServers(server)

		listener := bufconn.Listen(1 << 16)
		go func() { _ = server.Serve(listener) }()

		defer server.Stop()

		client, err := servicetemplate.NewClient(
			"passthrough:///bufconn",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)

		defer client.Close()

		callerConfig := trace.SpanContextConfig{
			TraceID:    trace.TraceID{1},
			SpanID:     trace.SpanID{1},
			TraceFlags: trace.FlagsSampled,
		}
		caller := trace.NewSpanContext(callerConfig)

		_, err = client.UnaryEcho(
			trace.ContextWithSpanContext(t.Context(), caller), &golibproto.UnaryEchoRequest{},
		)
		require.NoError(t, err)

		traceparent := <-received
		require.Len(t, traceparent, 1)
		require.Contains(t, traceparent[0], caller.TraceID().String())
	})
}

// clientConsumerModule and clientConsumerMain are a program outside this module that imports the
// client, as another service does.
const (
	clientConsumerModule = `module test/client

go 1.27.2

require github.com/a-novel/service-template v0.0.0

replace github.com/a-novel/service-template => %s
`
	clientConsumerMain = `package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicetemplate "github.com/a-novel/service-template/pkg/go"
)

func main() {
	request := &servicetemplate.ItemGetRequest{Id: "item"}
	if request.GetId() != "item" {
		panic("client request alias lost its generated methods")
	}

	client, err := servicetemplate.NewClient("localhost:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer client.Close()
}
`
)

// buildClientConsumer builds the client consumer program and returns the path of its binary.
func buildClientConsumer(t *testing.T) string {
	t.Helper()

	moduleRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	consumerDir := t.TempDir()
	goMod := fmt.Sprintf(clientConsumerModule, filepath.ToSlash(moduleRoot))
	require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "main.go"), []byte(clientConsumerMain), 0o600))

	build := exec.CommandContext(t.Context(), "go", "build", "-mod=mod", "-o", "consumer", ".")
	build.Dir = consumerDir

	build.Env = append(os.Environ(), "GOWORK=off")

	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))

	return filepath.Join(consumerDir, "consumer")
}
