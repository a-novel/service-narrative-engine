package servicetemplate

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

// Request, response, and entity types are re-exported from the service's generated
// protobuf definitions, so callers never import the service's internal packages.
type (
	StatusRequest  = templatev0.StatusRequest
	StatusResponse = templatev0.StatusResponse

	ItemCreateRequest  = templatev0.ItemCreateRequest
	ItemCreateResponse = templatev0.ItemCreateResponse
	ItemGetRequest     = templatev0.ItemGetRequest
	ItemGetResponse    = templatev0.ItemGetResponse
	ItemListRequest    = templatev0.ItemListRequest
	ItemListResponse   = templatev0.ItemListResponse
	ItemUpdateRequest  = templatev0.ItemUpdateRequest
	ItemUpdateResponse = templatev0.ItemUpdateResponse
	ItemDeleteRequest  = templatev0.ItemDeleteRequest
	ItemDeleteResponse = templatev0.ItemDeleteResponse

	Item = templatev0.Item
)

// A Client issues the service's gRPC calls, one method per RPC. Construct one
// with [NewClient] and call Close when finished to release the connection.
type Client interface {
	// UnaryEcho answers a fixed message without touching any dependency, so it checks the
	// connection alone.
	UnaryEcho(
		ctx context.Context, req *golibproto.UnaryEchoRequest, opts ...grpc.CallOption,
	) (*golibproto.UnaryEchoResponse, error)
	// Status reports the health of the service's dependencies. It returns Unavailable when a
	// dependency probe fails. Set a request deadline.
	Status(ctx context.Context, req *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	// ItemCreate creates an item and returns it.
	ItemCreate(ctx context.Context, req *ItemCreateRequest, opts ...grpc.CallOption) (*ItemCreateResponse, error)
	// ItemGet returns the item with the given ID.
	ItemGet(ctx context.Context, req *ItemGetRequest, opts ...grpc.CallOption) (*ItemGetResponse, error)
	// ItemList returns a page of items, newest first.
	ItemList(ctx context.Context, req *ItemListRequest, opts ...grpc.CallOption) (*ItemListResponse, error)
	// ItemUpdate replaces the name and description of an item and returns it.
	ItemUpdate(ctx context.Context, req *ItemUpdateRequest, opts ...grpc.CallOption) (*ItemUpdateResponse, error)
	// ItemDelete deletes an item and returns it.
	ItemDelete(ctx context.Context, req *ItemDeleteRequest, opts ...grpc.CallOption) (*ItemDeleteResponse, error)

	// Close releases the underlying gRPC connection. Call it once the client is
	// no longer needed.
	Close()
}

type client struct {
	golibproto.EchoServiceClient
	templatev0.StatusServiceClient
	templatev0.ItemCreateServiceClient
	templatev0.ItemGetServiceClient
	templatev0.ItemListServiceClient
	templatev0.ItemUpdateServiceClient
	templatev0.ItemDeleteServiceClient

	conn *grpc.ClientConn
}

func (c *client) Close() {
	_ = c.conn.Close()
}

// NewClient creates a [Client] for the service reachable at addr. The
// connection is established lazily on the first RPC. Dial options are forwarded
// to the underlying gRPC connection.
//
// Each call opens a client span and sends the caller's trace context, so the server's
// spans join the caller's trace.
func NewClient(addr string, opts ...grpc.DialOption) (Client, error) {
	opts = append([]grpc.DialOption{grpc.WithStatsHandler(otelgrpc.NewClientHandler())}, opts...)

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("new grpc client: %w", err)
	}

	return &client{
		EchoServiceClient:       golibproto.NewEchoServiceClient(conn),
		StatusServiceClient:     templatev0.NewStatusServiceClient(conn),
		ItemCreateServiceClient: templatev0.NewItemCreateServiceClient(conn),
		ItemGetServiceClient:    templatev0.NewItemGetServiceClient(conn),
		ItemListServiceClient:   templatev0.NewItemListServiceClient(conn),
		ItemUpdateServiceClient: templatev0.NewItemUpdateServiceClient(conn),
		ItemDeleteServiceClient: templatev0.NewItemDeleteServiceClient(conn),
		conn:                    conn,
	}, nil
}
