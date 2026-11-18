// Command rest serves the public REST API to the callers its service tokens allow. It refuses to start
// without a token issuer unless SERVICE_ISSUER_OPTIONAL opts a local stack out. cmd/migrations applies
// the database schema.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

	"github.com/a-novel/service-narrative-engine/internal/config"
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

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))
		defer cfg.Postgres.Close()
	}

	jsonKeysClient := lo.Must(servicejsonkeys.NewClient(
		cfg.Dependencies.ServiceJSONKeys.Address(), cfg.Dependencies.ServiceJSONKeys.DialOptions()...,
	))
	defer jsonKeysClient.Close()

	genaiClient := newGenaiClient(cfg.Dependencies.ServiceGenai)
	defer genaiClient.Close()

	router := lo.Must(newRouter(ctx, cfg, jsonKeysClient, genaiClient))

	log.Println("Starting REST server on :" + strconv.Itoa(cfg.Rest.Port))

	lo.Must0(httpf.ListenAndServe(ctx, cfg.Rest, router))
}

// newRouter builds the API behind the service-token gate, which wraps the whole router: a request
// reaches a route only with a token cfg accepts, and the mux reads the client IP header and counts
// rate limits for verified callers alone. The liveness route stays open to the kubelet. The gate
// refuses an empty issuer unless cfg opts a local stack out, so the server never starts open by
// mistake. The stream routes end their streams once ctx ends.
//
//nolint:maintidx // newRouter wires the whole dependency graph as one flat list, one line per dependency.
func newRouter(
	ctx context.Context, cfg config.App, jsonKeysClient servicejsonkeys.Client, genai genaiClient,
) (http.Handler, error) {
	withServiceToken, err := httpf.ServiceTokenGate(
		cfg.ServiceToken, httpf.NewPoolClient(httpf.PoolOptions{}), cfg.Logger, cfg.Rest.Liveness,
	)
	if err != nil {
		return nil, fmt.Errorf("service token gate: %w", err)
	}

	claimsVerifier, err := servicejsonkeys.NewClaimsVerifier[serviceauthentication.Claims](jsonKeysClient)
	if err != nil {
		return nil, fmt.Errorf("claims verifier: %w", err)
	}

	withAuth := serviceauthentication.NewAuthHandler(claimsVerifier, cfg.Permissions, cfg.Logger)

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoProjectInsert := dao.NewPgProjectInsert()
	daoProjectSelect := dao.NewPgProjectSelect()
	daoProjectOwned := dao.NewPgProjectOwned()
	daoProjectLock := dao.NewPgProjectLock()
	daoProjectBytes := dao.NewPgProjectBytes()
	daoProjectCount := dao.NewPgProjectCount()
	daoOwnerLock := dao.NewPgOwnerLock()
	daoProjectFieldsDrift := dao.NewPgProjectFieldsDrift()
	daoProjectEditsList := dao.NewPgProjectEditsList()
	daoProjectLogosDrift := dao.NewPgProjectLogosDrift()
	daoProjectUpdate := dao.NewPgProjectUpdate()
	daoProjectDelete := dao.NewPgProjectDelete()
	daoProjectLogoUpsert := dao.NewPgProjectLogoUpsert()
	daoProjectLogoSelect := dao.NewPgProjectLogoSelect()
	daoProjectLogoDelete := dao.NewPgProjectLogoDelete()
	daoBeatsInsert := dao.NewPgBeatsInsert()
	daoBeatList := dao.NewPgBeatList()
	daoBeatsUpdate := dao.NewPgBeatsUpdate()
	daoBeatMove := dao.NewPgBeatMove()
	daoBeatsDelete := dao.NewPgBeatsDelete()
	daoScenesInsert := dao.NewPgScenesInsert()
	daoSceneList := dao.NewPgSceneList()
	daoScenesUpdate := dao.NewPgScenesUpdate()
	daoSceneMove := dao.NewPgSceneMove()
	daoScenesDelete := dao.NewPgScenesDelete()
	daoSceneLength := dao.NewPgSceneLength()
	daoLayersInsert := dao.NewPgLayersInsert()
	daoLayerKind := dao.NewPgLayerKind()
	daoLayerList := dao.NewPgLayerList()
	daoLayersUpdate := dao.NewPgLayersUpdate()
	daoLayerMove := dao.NewPgLayerMove()
	daoLayersDelete := dao.NewPgLayersDelete()
	daoCellsUpsert := dao.NewPgCellsUpsert()
	daoCellList := dao.NewPgCellList()
	daoCellSummaryList := dao.NewPgCellSummaryList()
	daoCellsSelect := dao.NewPgCellsSelect()
	daoCellRecapUpdate := dao.NewPgCellRecapUpdate()
	daoBeatRecapUpdate := dao.NewPgBeatRecapUpdate()
	daoManuscriptSelect := dao.NewPgManuscriptSelect()
	transactor := postgres.NewTransactor(nil)
	projectReadGate := core.NewProjectReadGate(cfg.Limits.MaxConcurrentProjectReads)

	generationCaps, err := core.NewGenerationCapsPolicy(cfg.Permissions, cfg.Caps)
	if err != nil {
		return nil, fmt.Errorf("generation caps: %w", err)
	}

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceProjectCreate := core.NewProjectCreate(
		daoOwnerLock, daoProjectCount, daoProjectInsert, daoLayersInsert, transactor,
	)
	serviceProjectGet := core.NewProjectGet(daoProjectOwned, daoProjectSelect)
	serviceProjectFieldsDrift := core.NewProjectFieldsDrift(daoProjectFieldsDrift)
	serviceProjectEditsList := core.NewProjectEditsList(daoProjectEditsList)
	serviceProjectLogosDrift := core.NewProjectLogosDrift(daoProjectLogosDrift)
	serviceProjectUpdate := core.NewProjectUpdate(daoProjectLock, daoProjectUpdate, daoProjectBytes, transactor)
	serviceProjectDelete := core.NewProjectDelete(daoProjectOwned, daoProjectDelete)
	serviceProjectLogoUpdate := core.NewProjectLogoUpdate(daoProjectOwned, daoProjectLogoUpsert)
	serviceProjectLogoGet := core.NewProjectLogoGet(daoProjectOwned, daoProjectLogoSelect)
	serviceProjectLogoDelete := core.NewProjectLogoDelete(daoProjectOwned, daoProjectLogoDelete)
	serviceBeatsCreate := core.NewBeatsCreate(daoProjectLock, daoBeatsInsert, daoProjectBytes, transactor)
	serviceBeatList := core.NewBeatList(daoProjectOwned, daoBeatList)
	serviceBeatsUpdate := core.NewBeatsUpdate(daoProjectLock, daoBeatsUpdate, daoProjectBytes, transactor)
	serviceBeatMove := core.NewBeatMove(daoProjectLock, daoBeatMove, transactor)
	serviceBeatsDelete := core.NewBeatsDelete(daoProjectLock, daoBeatsDelete, transactor)
	serviceScenesCreate := core.NewScenesCreate(daoProjectLock, daoScenesInsert, daoProjectBytes, transactor)
	serviceSceneList := core.NewSceneList(daoProjectOwned, daoSceneList)
	serviceScenesUpdate := core.NewScenesUpdate(daoProjectLock, daoScenesUpdate, daoProjectBytes, transactor)
	serviceSceneMove := core.NewSceneMove(daoProjectLock, daoSceneMove, transactor)
	serviceScenesDelete := core.NewScenesDelete(daoProjectLock, daoScenesDelete, transactor)
	serviceLayersCreate := core.NewLayersCreate(daoProjectLock, daoLayersInsert, daoProjectBytes, transactor)
	serviceLayerList := core.NewLayerList(daoProjectOwned, daoLayerList)
	serviceLayersUpdate := core.NewLayersUpdate(daoProjectLock, daoLayersUpdate, daoProjectBytes, transactor)
	serviceLayerMove := core.NewLayerMove(daoProjectLock, daoLayerMove, transactor)
	serviceLayersDelete := core.NewLayersDelete(daoProjectLock, daoLayerKind, daoLayersDelete, transactor)
	serviceCellList := core.NewCellList(daoProjectOwned, daoCellList)
	serviceCellRecapWrite := core.NewCellRecapWrite(daoProjectLock, daoCellRecapUpdate, daoProjectBytes, transactor)
	serviceBeatRecapWrite := core.NewBeatRecapWrite(daoProjectLock, daoBeatRecapUpdate, daoProjectBytes, transactor)
	serviceCellsWrite := core.NewCellsWrite(
		daoProjectLock, daoLayerKind, daoCellsUpsert, daoSceneLength, daoProjectBytes, transactor,
	)
	serviceManuscriptExport := core.NewManuscriptExport(
		daoProjectOwned, daoLayerKind, daoProjectBytes, daoProjectSelect, daoManuscriptSelect, projectReadGate,
	)
	serviceProjectSnapshotRead := core.NewProjectSnapshotRead(
		daoProjectOwned, daoProjectBytes, daoProjectSelect, daoBeatList, daoSceneList, daoLayerList, daoCellSummaryList,
	)
	serviceGenerationPlan := core.NewGenerationPlan(cfg.Generations, daoCellsSelect)
	serviceGenerationSubmit := core.NewGenerationSubmit(
		serviceProjectSnapshotRead, serviceGenerationPlan, genai, projectReadGate, generationCaps,
	)
	serviceGenerationGet := core.NewGenerationGet(genai)
	serviceGenerationStream := core.NewGenerationStream(genai, serviceGenerationGet)
	serviceGenerationCancel := core.NewGenerationCancel(genai)
	serviceGenerationUsage := core.NewGenerationUsage(genai, generationCaps)
	serviceProjectStream := core.NewProjectStream(
		daoProjectOwned, daoProjectSelect, daoBeatList, daoSceneList, daoLayerList, daoCellList,
	)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerHealth := handlers.NewRestHealth(jsonKeysClient, genaiHealth(genai, cfg.Dependencies.ServiceGenai))
	handlerProjectCreate := handlers.NewRestProjectCreate(serviceProjectCreate, cfg.Logger)
	handlerProjectGet := handlers.NewRestProjectGet(serviceProjectGet, cfg.Logger)
	handlerProjectFieldsDrift := handlers.NewRestProjectFieldsDrift(serviceProjectFieldsDrift, cfg.Logger)
	handlerProjectEditsList := handlers.NewRestProjectEditsList(serviceProjectEditsList, cfg.Logger)
	handlerProjectLogosDrift := handlers.NewRestProjectLogosDrift(serviceProjectLogosDrift, cfg.Logger)
	handlerProjectUpdate := handlers.NewRestProjectUpdate(serviceProjectUpdate, cfg.Logger)
	handlerProjectDelete := handlers.NewRestProjectDelete(serviceProjectDelete, cfg.Logger)
	handlerProjectLogoUpdate := handlers.NewRestProjectLogoUpdate(serviceProjectLogoUpdate, cfg.Logger)
	handlerProjectLogoGet := handlers.NewRestProjectLogoGet(serviceProjectLogoGet, cfg.Logger)
	handlerProjectLogoDelete := handlers.NewRestProjectLogoDelete(serviceProjectLogoDelete, cfg.Logger)
	handlerBeatCreate := handlers.NewRestBeatCreate(serviceBeatsCreate, cfg.Logger)
	handlerBeatsCreate := handlers.NewRestBeatsCreate(serviceBeatsCreate, cfg.Logger)
	handlerBeatList := handlers.NewRestBeatList(serviceBeatList, cfg.Logger)
	handlerBeatUpdate := handlers.NewRestBeatUpdate(serviceBeatsUpdate, cfg.Logger)
	handlerBeatsUpdate := handlers.NewRestBeatsUpdate(serviceBeatsUpdate, cfg.Logger)
	handlerBeatMove := handlers.NewRestBeatMove(serviceBeatMove, cfg.Logger)
	handlerBeatDelete := handlers.NewRestBeatDelete(serviceBeatsDelete, cfg.Logger)
	handlerBeatsDelete := handlers.NewRestBeatsDelete(serviceBeatsDelete, cfg.Logger)
	handlerSceneCreate := handlers.NewRestSceneCreate(serviceScenesCreate, cfg.Logger)
	handlerScenesCreate := handlers.NewRestScenesCreate(serviceScenesCreate, cfg.Logger)
	handlerSceneList := handlers.NewRestSceneList(serviceSceneList, cfg.Logger)
	handlerSceneUpdate := handlers.NewRestSceneUpdate(serviceScenesUpdate, cfg.Logger)
	handlerScenesUpdate := handlers.NewRestScenesUpdate(serviceScenesUpdate, cfg.Logger)
	handlerSceneMove := handlers.NewRestSceneMove(serviceSceneMove, cfg.Logger)
	handlerSceneDelete := handlers.NewRestSceneDelete(serviceScenesDelete, cfg.Logger)
	handlerScenesDelete := handlers.NewRestScenesDelete(serviceScenesDelete, cfg.Logger)
	handlerLayerCreate := handlers.NewRestLayerCreate(serviceLayersCreate, cfg.Logger)
	handlerLayersCreate := handlers.NewRestLayersCreate(serviceLayersCreate, cfg.Logger)
	handlerLayerList := handlers.NewRestLayerList(serviceLayerList, cfg.Logger)
	handlerLayerUpdate := handlers.NewRestLayerUpdate(serviceLayersUpdate, cfg.Logger)
	handlerLayersUpdate := handlers.NewRestLayersUpdate(serviceLayersUpdate, cfg.Logger)
	handlerLayerMove := handlers.NewRestLayerMove(serviceLayerMove, cfg.Logger)
	handlerLayerDelete := handlers.NewRestLayerDelete(serviceLayersDelete, cfg.Logger)
	handlerLayersDelete := handlers.NewRestLayersDelete(serviceLayersDelete, cfg.Logger)
	handlerCellList := handlers.NewRestCellList(serviceCellList, cfg.Logger)
	handlerCellWrite := handlers.NewRestCellWrite(serviceCellsWrite, cfg.Logger)
	handlerCellsWrite := handlers.NewRestCellsWrite(serviceCellsWrite, cfg.Logger)
	handlerCellRecapWrite := handlers.NewRestCellRecapWrite(serviceCellRecapWrite, cfg.Logger)
	handlerBeatRecapWrite := handlers.NewRestBeatRecapWrite(serviceBeatRecapWrite, cfg.Logger)
	handlerManuscriptExport := handlers.NewRestManuscriptExport(serviceManuscriptExport, cfg.Logger)
	handlerGenerationSubmit := handlers.NewRestGenerationSubmit(serviceGenerationSubmit, cfg.Logger)
	handlerGenerationGet := handlers.NewRestGenerationGet(serviceGenerationGet, cfg.Logger)
	handlerGenerationStream := handlers.NewRestGenerationStream(serviceGenerationStream, cfg.Logger)
	handlerGenerationCancel := handlers.NewRestGenerationCancel(serviceGenerationCancel, cfg.Logger)
	handlerGenerationUsage := handlers.NewRestGenerationUsage(serviceGenerationUsage, cfg.Logger)
	handlerProjectStream := handlers.NewRestProjectStream(serviceProjectStream, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewMux(cfg.Rest, cfg.RestLogger, cfg.App.DowntimeStart)
	limit := handlers.NewRestLimits(cfg.Rest, cfg.Limits)

	router.Route("/v1", func(api chi.Router) {
		// The router's body limit fits a batch; a route that writes one item, or compares what a client holds,
		// caps its body lower.
		item := api.With(middleware.RequestSize(cfg.Limits.MaxItemRequestSize))
		// A stream outlives the request timeout, and ends at shutdown: its client resumes on another
		// instance. One cap counts a caller's open streams across both stream routes.
		stream := api.With(httpf.Streaming(cfg.Limits.MaxStreamDuration), httpf.EndOnShutdown(ctx))

		api.Get("/ping", httpf.Ping)
		api.Get("/healthcheck", handlerHealth)

		withAuth(item, config.PermissionProjectWrite).Post("/projects", handlerProjectCreate.ServeHTTP)
		withAuth(item, config.PermissionProjectRead).Post("/projects/fields/drift", handlerProjectFieldsDrift.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/projects/edits", handlerProjectEditsList.ServeHTTP)
		withAuth(item, config.PermissionProjectRead).Post("/projects/logos/drift", handlerProjectLogosDrift.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/project", handlerProjectGet.ServeHTTP)
		withAuth(stream, config.PermissionProjectRead).With(limit.Stream).
			Get("/project/stream", handlerProjectStream.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/project", handlerProjectUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/project", handlerProjectDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/project/logo", handlerProjectLogoUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/project/logo", handlerProjectLogoGet.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/project/logo", handlerProjectLogoDelete.ServeHTTP)

		withAuth(item, config.PermissionProjectWrite).Post("/beat", handlerBeatCreate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/beat", handlerBeatUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/beat", handlerBeatDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Post("/beat/move", handlerBeatMove.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/beats", handlerBeatList.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Post("/beats", handlerBeatsCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Put("/beats", handlerBeatsUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/beats", handlerBeatsDelete.ServeHTTP)

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

		withAuth(item, config.PermissionProjectWrite).Put("/cell/recap", handlerCellRecapWrite.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/beat/recap", handlerBeatRecapWrite.ServeHTTP)

		withAuth(api, config.PermissionProjectRead).Get("/manuscript", handlerManuscriptExport.ServeHTTP)

		// A submit carries its overrides, so it takes a batch's body limit.
		withAuth(api, config.PermissionProjectWrite).With(limit.Generation).
			Post("/generation", handlerGenerationSubmit.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).With(limit.GenerationRead).
			Get("/generation", handlerGenerationGet.ServeHTTP)
		withAuth(stream, config.PermissionProjectRead).With(limit.GenerationStream).
			Get("/generation/stream", handlerGenerationStream.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).Delete("/generation", handlerGenerationCancel.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/generation/usage", handlerGenerationUsage.ServeHTTP)
	})

	return withServiceToken(router), nil
}
