package servicetemplate

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-template/internal/handlers/protogen"
)

type (
	StatusRequest  = protogen.StatusRequest
	StatusResponse = protogen.StatusResponse

	ItemCreateRequest  = protogen.ItemCreateRequest
	ItemCreateResponse = protogen.ItemCreateResponse
	ItemGetRequest     = protogen.ItemGetRequest
	ItemGetResponse    = protogen.ItemGetResponse
	ItemListRequest    = protogen.ItemListRequest
	ItemListResponse   = protogen.ItemListResponse
	ItemUpdateRequest  = protogen.ItemUpdateRequest
	ItemUpdateResponse = protogen.ItemUpdateResponse
	ItemDeleteRequest  = protogen.ItemDeleteRequest
	ItemDeleteResponse = protogen.ItemDeleteResponse

	Item = protogen.Item
)

// Client is the gRPC client interface for the service.
type Client interface {
	UnaryEcho(
		ctx context.Context, req *golibproto.UnaryEchoRequest, opts ...grpc.CallOption,
	) (*golibproto.UnaryEchoResponse, error)
	Status(ctx context.Context, req *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)

	ItemCreate(ctx context.Context, req *ItemCreateRequest, opts ...grpc.CallOption) (*ItemCreateResponse, error)
	ItemGet(ctx context.Context, req *ItemGetRequest, opts ...grpc.CallOption) (*ItemGetResponse, error)
	ItemList(ctx context.Context, req *ItemListRequest, opts ...grpc.CallOption) (*ItemListResponse, error)
	ItemUpdate(ctx context.Context, req *ItemUpdateRequest, opts ...grpc.CallOption) (*ItemUpdateResponse, error)
	ItemDelete(ctx context.Context, req *ItemDeleteRequest, opts ...grpc.CallOption) (*ItemDeleteResponse, error)

	Close()
}

type client struct {
	golibproto.EchoServiceClient
	protogen.StatusServiceClient
	protogen.ItemCreateServiceClient
	protogen.ItemGetServiceClient
	protogen.ItemListServiceClient
	protogen.ItemUpdateServiceClient
	protogen.ItemDeleteServiceClient

	conn *grpc.ClientConn
}

func (c *client) Close() {
	_ = c.conn.Close()
}

func NewClient(addr string, opts ...grpc.DialOption) (Client, error) {
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("new grpc client: %w", err)
	}

	return &client{
		EchoServiceClient:       golibproto.NewEchoServiceClient(conn),
		StatusServiceClient:     protogen.NewStatusServiceClient(conn),
		ItemCreateServiceClient: protogen.NewItemCreateServiceClient(conn),
		ItemGetServiceClient:    protogen.NewItemGetServiceClient(conn),
		ItemListServiceClient:   protogen.NewItemListServiceClient(conn),
		ItemUpdateServiceClient: protogen.NewItemUpdateServiceClient(conn),
		ItemDeleteServiceClient: protogen.NewItemDeleteServiceClient(conn),
		conn:                    conn,
	}, nil
}
