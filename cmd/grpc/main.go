// Command grpc serves the private gRPC API, for other services on the internal network. cmd/rest
// serves the same domain over HTTP, and cmd/migrations applies the database schema.
package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/samber/lo"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-template/internal/config"
	"github.com/a-novel/service-template/internal/config/env"
	"github.com/a-novel/service-template/internal/core"
	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/handlers"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
)

func main() {
	cfg := config.AppPresetDefault

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectID == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

		database := lo.Must(cfg.Postgres.DB(ctx))
		defer closeDatabase(database)
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoItemCreate := dao.NewPgItemCreate()
	daoItemGet := dao.NewPgItemGet()
	daoItemList := dao.NewPgItemList()
	daoItemUpdate := dao.NewPgItemUpdate()
	daoItemDelete := dao.NewPgItemDelete()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceItemCreate := core.NewItemCreate(daoItemCreate)
	serviceItemGet := core.NewItemGet(daoItemGet)
	serviceItemList := core.NewItemList(daoItemList)
	serviceItemUpdate := core.NewItemUpdate(daoItemUpdate)
	serviceItemDelete := core.NewItemDelete(daoItemDelete)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerStatus := handlers.NewGrpcStatus()
	handlerItemCreate := handlers.NewGrpcItemCreate(serviceItemCreate)
	handlerItemGet := handlers.NewGrpcItemGet(serviceItemGet)
	handlerItemList := handlers.NewGrpcItemList(serviceItemList)
	handlerItemUpdate := handlers.NewGrpcItemUpdate(serviceItemUpdate)
	handlerItemDelete := handlers.NewGrpcItemDelete(serviceItemDelete)

	// =================================================================================================================
	// SERVER
	// =================================================================================================================

	ctxInterceptor := func(rpCtx context.Context) context.Context {
		return postgres.TransferContext(ctx, rpCtx)
	}

	server := grpcf.NewServer(ctx, cfg.Otel, cfg.GrpcLogger, cfg.App.DowntimeStart, ctxInterceptor)

	healthcheck := grpcf.RegisterEchoServers(server)
	healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	go func() {
		<-ctx.Done()
		healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}()

	templatev0.RegisterStatusServiceServer(server, handlerStatus)
	templatev0.RegisterItemCreateServiceServer(server, handlerItemCreate)
	templatev0.RegisterItemGetServiceServer(server, handlerItemGet)
	templatev0.RegisterItemListServiceServer(server, handlerItemList)
	templatev0.RegisterItemUpdateServiceServer(server, handlerItemUpdate)
	templatev0.RegisterItemDeleteServiceServer(server, handlerItemDelete)

	reflection.Register(server)

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting gRPC server on :" + strconv.Itoa(cfg.Grpc.Port))

	lo.Must0(grpcf.ListenAndServe(ctx, cfg.Grpc, server))
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
