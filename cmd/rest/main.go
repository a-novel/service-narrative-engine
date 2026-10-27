// Command rest serves the public REST API, for any HTTP client. cmd/migrations applies the
// database schema.
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

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/sentryf"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/env"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func main() {
	cfg := config.AppPresetDefault

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	lo.Must0(sentryf.Init(cfg.Sentry))

	defer sentryf.Flush(config.SentryFlushTimeout)

	if env.OTLPEndpoint == "" {
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

	handlerPing := handlers.NewRestPing()
	handlerHealth := handlers.NewRestHealth()
	handlerItemCreate := handlers.NewRestItemCreate(serviceItemCreate, cfg.Logger)
	handlerItemGet := handlers.NewRestItemGet(serviceItemGet, cfg.Logger)
	handlerItemList := handlers.NewRestItemList(serviceItemList, cfg.Logger)
	handlerItemUpdate := handlers.NewRestItemUpdate(serviceItemUpdate, cfg.Logger)
	handlerItemDelete := handlers.NewRestItemDelete(serviceItemDelete, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewRouter(cfg.Rest, cfg.Otel, cfg.RestLogger, cfg.App.DowntimeStart)

	router.Route("/v0", func(api chi.Router) {
		api.Get("/ping", handlerPing.ServeHTTP)
		api.Get("/healthcheck", handlerHealth.ServeHTTP)
		api.Post("/items", handlerItemCreate.ServeHTTP)
		api.Get("/items", handlerItemList.ServeHTTP)
		api.Get("/item", handlerItemGet.ServeHTTP)
		api.Put("/item", handlerItemUpdate.ServeHTTP)
		api.Delete("/item", handlerItemDelete.ServeHTTP)
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting REST server on :" + strconv.Itoa(cfg.Rest.Port))

	lo.Must0(httpf.ListenAndServe(ctx, cfg.Rest, router))
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
