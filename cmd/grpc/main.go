package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/samber/lo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-template/internal/config"
	"github.com/a-novel/service-template/internal/config/env"
	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/handlers"
	"github.com/a-novel/service-template/internal/handlers/protogen"
	"github.com/a-novel/service-template/internal/services"
)

// Runs the main gRPC server.
func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	ctx = lo.Must(postgres.NewContext(ctx, config.PostgresPresetDefault))

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	repositoryItemCreate := dao.NewItemCreate()
	repositoryItemGet := dao.NewItemGet()
	repositoryItemList := dao.NewItemList()
	repositoryItemUpdate := dao.NewItemUpdate()
	repositoryItemDelete := dao.NewItemDelete()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceItemCreate := services.NewItemCreate(repositoryItemCreate)
	serviceItemGet := services.NewItemGet(repositoryItemGet)
	serviceItemList := services.NewItemList(repositoryItemList)
	serviceItemUpdate := services.NewItemUpdate(repositoryItemUpdate)
	serviceItemDelete := services.NewItemDelete(repositoryItemDelete)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerStatus := handlers.NewGrpcStatus()
	handlerItemCreate := handlers.NewItemCreate(serviceItemCreate)
	handlerItemGet := handlers.NewItemGet(serviceItemGet)
	handlerItemList := handlers.NewItemList(serviceItemList)
	handlerItemUpdate := handlers.NewItemUpdate(serviceItemUpdate)
	handlerItemDelete := handlers.NewItemDelete(serviceItemDelete)

	// =================================================================================================================
	// SERVER
	// =================================================================================================================

	ctxInterceptor := func(rpCtx context.Context) context.Context {
		return postgres.TransferContext(ctx, rpCtx)
	}

	listenerConfig := new(net.ListenConfig)
	listener := lo.Must(listenerConfig.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", cfg.Grpc.Port)))
	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		cfg.Otel.RpcInterceptor(),
		grpc.ChainUnaryInterceptor(
			grpcf.BaseContextUnaryInterceptor(ctxInterceptor),
			cfg.Logger.UnaryInterceptor(),
			cfg.Logger.PanicUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcf.BaseContextStreamInterceptor(ctxInterceptor),
			cfg.Logger.StreamInterceptor(),
			cfg.Logger.PanicStreamInterceptor(),
		),
	)

	grpcf.SetEchoServersContext(ctx, server, cfg.Grpc.Ping)

	protogen.RegisterStatusServiceServer(server, handlerStatus)
	protogen.RegisterItemCreateServiceServer(server, handlerItemCreate)
	protogen.RegisterItemGetServiceServer(server, handlerItemGet)
	protogen.RegisterItemListServiceServer(server, handlerItemList)
	protogen.RegisterItemUpdateServiceServer(server, handlerItemUpdate)
	protogen.RegisterItemDeleteServiceServer(server, handlerItemDelete)

	reflection.Register(server)

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting gRPC server on :" + strconv.Itoa(cfg.Grpc.Port))

	go func() {
		err := server.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	server.GracefulStop()
}
