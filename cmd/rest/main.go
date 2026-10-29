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
	"github.com/go-chi/chi/v5/middleware"
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
	daoProjectOwned := dao.NewPgProjectOwned()
	daoProjectLock := dao.NewPgProjectLock()
	daoProjectList := dao.NewPgProjectList()
	daoProjectUpdate := dao.NewPgProjectUpdate()
	daoProjectDelete := dao.NewPgProjectDelete()
	daoSceneInsert := dao.NewPgSceneInsert()
	daoScenesInsert := dao.NewPgScenesInsert()
	daoSceneList := dao.NewPgSceneList()
	daoSceneUpdate := dao.NewPgSceneUpdate()
	daoScenesUpdate := dao.NewPgScenesUpdate()
	daoSceneMove := dao.NewPgSceneMove()
	daoSceneDelete := dao.NewPgSceneDelete()
	daoScenesDelete := dao.NewPgScenesDelete()
	daoLayerInsert := dao.NewPgLayerInsert()
	daoLayersInsert := dao.NewPgLayersInsert()
	daoLayerKind := dao.NewPgLayerKind()
	daoLayerList := dao.NewPgLayerList()
	daoLayerUpdate := dao.NewPgLayerUpdate()
	daoLayersUpdate := dao.NewPgLayersUpdate()
	daoLayerMove := dao.NewPgLayerMove()
	daoLayerDelete := dao.NewPgLayerDelete()
	daoLayersDelete := dao.NewPgLayersDelete()
	daoCellUpsert := dao.NewPgCellUpsert()
	daoCellsUpsert := dao.NewPgCellsUpsert()
	daoCellList := dao.NewPgCellList()
	transactor := postgres.NewTransactor(nil)

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceProjectCreate := core.NewProjectCreate(daoProjectInsert, daoLayersInsert, transactor)
	serviceProjectGet := core.NewProjectGet(daoProjectOwned, daoProjectSelect)
	serviceProjectList := core.NewProjectList(daoProjectList)
	serviceProjectUpdate := core.NewProjectUpdate(daoProjectOwned, daoProjectUpdate)
	serviceProjectDelete := core.NewProjectDelete(daoProjectOwned, daoProjectDelete)
	serviceSceneCreate := core.NewSceneCreate(daoProjectLock, daoSceneInsert, transactor)
	serviceScenesCreate := core.NewScenesCreate(daoProjectLock, daoScenesInsert, transactor)
	serviceSceneList := core.NewSceneList(daoProjectOwned, daoSceneList)
	serviceSceneUpdate := core.NewSceneUpdate(daoProjectOwned, daoSceneUpdate)
	serviceScenesUpdate := core.NewScenesUpdate(daoProjectOwned, daoScenesUpdate)
	serviceSceneMove := core.NewSceneMove(daoProjectLock, daoSceneMove, transactor)
	serviceSceneDelete := core.NewSceneDelete(daoProjectLock, daoSceneDelete, transactor)
	serviceScenesDelete := core.NewScenesDelete(daoProjectLock, daoScenesDelete, transactor)
	serviceLayerCreate := core.NewLayerCreate(daoProjectLock, daoLayerInsert, transactor)
	serviceLayersCreate := core.NewLayersCreate(daoProjectLock, daoLayersInsert, transactor)
	serviceLayerList := core.NewLayerList(daoProjectOwned, daoLayerList)
	serviceLayerUpdate := core.NewLayerUpdate(daoProjectOwned, daoLayerUpdate)
	serviceLayersUpdate := core.NewLayersUpdate(daoProjectOwned, daoLayersUpdate)
	serviceLayerMove := core.NewLayerMove(daoProjectLock, daoLayerMove, transactor)
	serviceLayerDelete := core.NewLayerDelete(daoProjectLock, daoLayerKind, daoLayerDelete, transactor)
	serviceLayersDelete := core.NewLayersDelete(daoProjectLock, daoLayerKind, daoLayersDelete, transactor)
	serviceCellList := core.NewCellList(daoProjectOwned, daoCellList)
	serviceCellWrite := core.NewCellWrite(daoProjectOwned, daoLayerKind, daoCellUpsert)
	serviceCellsWrite := core.NewCellsWrite(daoProjectOwned, daoLayerKind, daoCellsUpsert)

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
	handlerScenesCreate := handlers.NewRestScenesCreate(serviceScenesCreate, cfg.Logger)
	handlerSceneList := handlers.NewRestSceneList(serviceSceneList, cfg.Logger)
	handlerSceneUpdate := handlers.NewRestSceneUpdate(serviceSceneUpdate, cfg.Logger)
	handlerScenesUpdate := handlers.NewRestScenesUpdate(serviceScenesUpdate, cfg.Logger)
	handlerSceneMove := handlers.NewRestSceneMove(serviceSceneMove, cfg.Logger)
	handlerSceneDelete := handlers.NewRestSceneDelete(serviceSceneDelete, cfg.Logger)
	handlerScenesDelete := handlers.NewRestScenesDelete(serviceScenesDelete, cfg.Logger)
	handlerLayerCreate := handlers.NewRestLayerCreate(serviceLayerCreate, cfg.Logger)
	handlerLayersCreate := handlers.NewRestLayersCreate(serviceLayersCreate, cfg.Logger)
	handlerLayerList := handlers.NewRestLayerList(serviceLayerList, cfg.Logger)
	handlerLayerUpdate := handlers.NewRestLayerUpdate(serviceLayerUpdate, cfg.Logger)
	handlerLayersUpdate := handlers.NewRestLayersUpdate(serviceLayersUpdate, cfg.Logger)
	handlerLayerMove := handlers.NewRestLayerMove(serviceLayerMove, cfg.Logger)
	handlerLayerDelete := handlers.NewRestLayerDelete(serviceLayerDelete, cfg.Logger)
	handlerLayersDelete := handlers.NewRestLayersDelete(serviceLayersDelete, cfg.Logger)
	handlerCellList := handlers.NewRestCellList(serviceCellList, cfg.Logger)
	handlerCellWrite := handlers.NewRestCellWrite(serviceCellWrite, cfg.Logger)
	handlerCellsWrite := handlers.NewRestCellsWrite(serviceCellsWrite, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewRouter(cfg.Rest, cfg.Otel, cfg.RestLogger, cfg.App.DowntimeStart)

	router.Route("/v1", func(api chi.Router) {
		// The router's body limit fits a batch; a route that writes one item caps its body lower.
		item := api.With(middleware.RequestSize(cfg.Limits.ItemMaxRequestSize))

		api.Get("/ping", handlerPing.ServeHTTP)
		api.Get("/healthcheck", handlerHealth.ServeHTTP)

		withAuth(item, config.PermissionProjectWrite).Post("/projects", handlerProjectCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/projects", handlerProjectList.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/project", handlerProjectGet.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/project", handlerProjectUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/project", handlerProjectDelete.ServeHTTP)

		withAuth(item, config.PermissionProjectWrite).Post("/scene", handlerSceneCreate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/scene", handlerSceneUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/scene", handlerSceneDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Post("/scene/move", handlerSceneMove.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/scenes", handlerSceneList.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Post("/scenes", handlerScenesCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/scenes", handlerScenesUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/scenes", handlerScenesDelete.ServeHTTP)

		withAuth(item, config.PermissionProjectWrite).Post("/layer", handlerLayerCreate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/layer", handlerLayerUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/layer", handlerLayerDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Post("/layer/move", handlerLayerMove.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/layers", handlerLayerList.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Post("/layers", handlerLayersCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/layers", handlerLayersUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/layers", handlerLayersDelete.ServeHTTP)

		withAuth(item, config.PermissionProjectWrite).Put("/cell", handlerCellWrite.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/cells", handlerCellList.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/cells", handlerCellsWrite.ServeHTTP)
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
