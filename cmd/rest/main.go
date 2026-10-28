// Command rest serves the public REST API, for any HTTP client. cmd/migrations applies the
// database schema.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

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
	// CLIENTS
	// =================================================================================================================

	jsonKeysClient := lo.Must(servicejsonkeys.NewClient(
		fmt.Sprintf("%s:%d", cfg.Dependencies.ServiceJsonKeysHost, cfg.Dependencies.ServiceJsonKeysPort),
		lo.Must(cfg.Dependencies.ServiceJsonKeysCredentials.Options(ctx))...,
	))
	defer jsonKeysClient.Close()

	claimsVerifier := lo.Must(servicejsonkeys.NewClaimsVerifier[serviceauthentication.Claims](jsonKeysClient))
	withAuth := serviceauthentication.NewAuthHandler(claimsVerifier, cfg.Permissions, cfg.Logger)

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoProjectInsert := dao.NewPgProjectInsert()
	daoProjectSelect := dao.NewPgProjectSelect()
	daoProjectLock := dao.NewPgProjectLock()
	daoProjectList := dao.NewPgProjectList()
	daoProjectCount := dao.NewPgProjectCount()
	daoProjectUpdate := dao.NewPgProjectUpdate()
	daoProjectDelete := dao.NewPgProjectDelete()
	daoSceneInsert := dao.NewPgSceneInsert()
	daoSceneSelect := dao.NewPgSceneSelect()
	daoSceneList := dao.NewPgSceneList()
	daoSceneUpdate := dao.NewPgSceneUpdate()
	daoSceneDelete := dao.NewPgSceneDelete()
	daoSceneReorder := dao.NewPgSceneReorder()
	daoLayerInsert := dao.NewPgLayerInsert()
	daoLayerSelect := dao.NewPgLayerSelect()
	daoLayerList := dao.NewPgLayerList()
	daoLayerUpdate := dao.NewPgLayerUpdate()
	daoLayerDelete := dao.NewPgLayerDelete()
	daoLayerReorder := dao.NewPgLayerReorder()
	daoCellUpsert := dao.NewPgCellUpsert()
	daoCellList := dao.NewPgCellList()
	daoCellCount := dao.NewPgCellCount()
	transactor := postgres.NewTransactor(nil)

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceProjectCreate := core.NewProjectCreate(
		daoProjectInsert, daoLayerInsert, daoSceneInsert, transactor, cfg.Composer,
	)
	serviceProjectGet := core.NewProjectGet(daoProjectSelect, daoSceneList, daoLayerList)
	serviceProjectList := core.NewProjectList(daoProjectList, daoProjectCount)
	serviceProjectUpdate := core.NewProjectUpdate(daoProjectUpdate)
	serviceProjectDelete := core.NewProjectDelete(daoProjectDelete)
	serviceSceneCreate := core.NewSceneCreate(daoProjectLock, daoSceneList, daoSceneInsert, daoSceneReorder, transactor)
	serviceSceneUpdate := core.NewSceneUpdate(daoSceneUpdate)
	serviceSceneMove := core.NewSceneMove(daoSceneSelect, daoProjectLock, daoSceneList, daoSceneReorder, transactor)
	serviceSceneDelete := core.NewSceneDelete(
		daoSceneSelect, daoProjectLock, daoSceneDelete, daoSceneList, daoSceneReorder, transactor,
	)
	serviceLayerCreate := core.NewLayerCreate(daoProjectLock, daoLayerList, daoLayerInsert, daoLayerReorder, transactor)
	serviceLayerUpdate := core.NewLayerUpdate(daoLayerUpdate)
	serviceLayerMove := core.NewLayerMove(daoLayerSelect, daoProjectLock, daoLayerList, daoLayerReorder, transactor)
	serviceLayerDelete := core.NewLayerDelete(
		daoLayerSelect, daoProjectLock, daoLayerDelete, daoLayerList, daoLayerReorder, transactor,
	)
	serviceCellList := core.NewCellList(daoProjectSelect, daoCellList, daoCellCount)
	serviceCellWrite := core.NewCellWrite(daoSceneSelect, daoLayerSelect, daoCellUpsert, cfg.Composer)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewRestPing()
	handlerHealth := handlers.NewRestHealth(jsonKeysClient)
	handlerProjectCreate := handlers.NewRestProjectCreate(serviceProjectCreate, cfg.Logger)
	handlerProjectGet := handlers.NewRestProjectGet(serviceProjectGet, cfg.Logger)
	handlerProjectList := handlers.NewRestProjectList(serviceProjectList, cfg.Logger)
	handlerProjectUpdate := handlers.NewRestProjectUpdate(serviceProjectUpdate, cfg.Logger)
	handlerProjectDelete := handlers.NewRestProjectDelete(serviceProjectDelete, cfg.Logger)
	handlerSceneCreate := handlers.NewRestSceneCreate(serviceSceneCreate, cfg.Logger)
	handlerSceneUpdate := handlers.NewRestSceneUpdate(serviceSceneUpdate, cfg.Logger)
	handlerSceneMove := handlers.NewRestSceneMove(serviceSceneMove, cfg.Logger)
	handlerSceneDelete := handlers.NewRestSceneDelete(serviceSceneDelete, cfg.Logger)
	handlerLayerCreate := handlers.NewRestLayerCreate(serviceLayerCreate, cfg.Logger)
	handlerLayerUpdate := handlers.NewRestLayerUpdate(serviceLayerUpdate, cfg.Logger)
	handlerLayerMove := handlers.NewRestLayerMove(serviceLayerMove, cfg.Logger)
	handlerLayerDelete := handlers.NewRestLayerDelete(serviceLayerDelete, cfg.Logger)
	handlerCellList := handlers.NewRestCellList(serviceCellList, cfg.Logger)
	handlerCellWrite := handlers.NewRestCellWrite(serviceCellWrite, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewRouter(cfg.Rest, cfg.Otel, cfg.RestLogger, cfg.App.DowntimeStart)

	router.Route("/v1", func(api chi.Router) {
		api.Get("/ping", handlerPing.ServeHTTP)
		api.Get("/healthcheck", handlerHealth.ServeHTTP)

		withAuth(api, config.PermissionProjectWrite).Post("/projects", handlerProjectCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/projects", handlerProjectList.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/project", handlerProjectGet.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/project", handlerProjectUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/project", handlerProjectDelete.ServeHTTP)

		withAuth(api, config.PermissionProjectWrite).Post("/scenes", handlerSceneCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/scene", handlerSceneUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Post("/scene/move", handlerSceneMove.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/scene", handlerSceneDelete.ServeHTTP)

		withAuth(api, config.PermissionProjectWrite).Post("/layers", handlerLayerCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/layer", handlerLayerUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Post("/layer/move", handlerLayerMove.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/layer", handlerLayerDelete.ServeHTTP)

		withAuth(api, config.PermissionProjectRead).Get("/cells", handlerCellList.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/cell", handlerCellWrite.ServeHTTP)
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
