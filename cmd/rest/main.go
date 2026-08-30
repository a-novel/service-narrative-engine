// Command rest wires the service layers together and runs the public REST server. It builds the
// DAO, core, and handler stack, mounts the routes, and serves until a termination signal triggers
// a graceful shutdown.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/env"
	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	// =================================================================================================================
	// CLIENTS
	// =================================================================================================================

	genaiClient := lo.Must(servicegenai.NewClient(
		fmt.Sprintf("%s:%d", cfg.Dependencies.ServiceGenAIHost, cfg.Dependencies.ServiceGenAIPort),
		lo.Must(cfg.Dependencies.ServiceGenAICredentials.Options(ctx))...,
	))
	defer genaiClient.Close()

	jsonKeysClient := lo.Must(servicejsonkeys.NewClient(
		fmt.Sprintf(
			"%s:%d",
			cfg.Dependencies.ServiceJsonKeysHost,
			cfg.Dependencies.ServiceJsonKeysPort,
		),
		lo.Must(cfg.Dependencies.ServiceJsonKeysCredentials.Options(ctx))...,
	))
	defer jsonKeysClient.Close()

	claimsVerifier := lo.Must(
		servicejsonkeys.NewClaimsVerifier[serviceauthentication.Claims](jsonKeysClient),
	)
	withAuth := serviceauthentication.NewAuthHandler(claimsVerifier, cfg.Permissions, cfg.Logger)

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	transactor := postgres.NewTransactor(nil)
	projectSelectDao := dao.NewPgProjectSelect()
	ideaInsertDao := dao.NewPgIdeaInsert()
	ideaSelectDao := dao.NewPgIdeaSelect()
	ideaVersionInsertDao := dao.NewPgIdeaVersionInsert()
	ideaVersionListDao := dao.NewPgIdeaVersionList()
	stepValueInsertDao := dao.NewPgStepValueInsert()
	stepValueCurrentListDao := dao.NewPgStepValueCurrentList()
	stepValueListDao := dao.NewPgStepValueList()
	manuscriptInsertDao := dao.NewPgManuscriptInsert()
	manuscriptSelectDao := dao.NewPgManuscriptSelect()
	manuscriptListDao := dao.NewPgManuscriptList()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	projectAccess := core.NewProjectAccess(projectSelectDao)
	projectGet := core.NewProjectGet(
		projectAccess,
		ideaSelectDao,
		stepValueCurrentListDao,
		manuscriptSelectDao,
	)
	ideaCreate := core.NewIdeaCreate(ideaInsertDao)
	ideaVersionCreate := core.NewIdeaVersionCreate(
		projectAccess,
		ideaVersionInsertDao,
		transactor,
	)
	ideaHistory := core.NewIdeaHistory(projectAccess, ideaVersionListDao)
	stepValueCreate := core.NewStepValueCreate(projectAccess, stepValueInsertDao, transactor)
	stepValueHistory := core.NewStepValueHistory(projectAccess, stepValueListDao)
	manuscriptCreate := core.NewManuscriptCreate(projectAccess, manuscriptInsertDao, transactor)
	manuscriptHistory := core.NewManuscriptHistory(projectAccess, manuscriptListDao)
	generationSubmit := core.NewGenerationSubmit(projectAccess, genaiClient)
	generationGet := core.NewGenerationGet(projectAccess, genaiClient)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewPing()
	handlerHealth := handlers.NewRestHealth(jsonKeysClient, genaiClient)
	handlerProjectCreate := handlers.NewRestProjectCreate(ideaCreate, cfg.Logger)
	handlerProjectGet := handlers.NewRestProjectGet(projectGet, cfg.Logger)
	handlerIdeaVersionCreate := handlers.NewRestIdeaVersionCreate(ideaVersionCreate, cfg.Logger)
	handlerIdeaHistory := handlers.NewRestIdeaHistory(ideaHistory, cfg.Logger)
	handlerStepValueCreate := handlers.NewRestStepValueCreate(stepValueCreate, cfg.Logger)
	handlerStepValueHistory := handlers.NewRestStepValueHistory(stepValueHistory, cfg.Logger)
	handlerManuscriptCreate := handlers.NewRestManuscriptCreate(manuscriptCreate, cfg.Logger)
	handlerManuscriptHistory := handlers.NewRestManuscriptHistory(manuscriptHistory, cfg.Logger)
	handlerGenerationSubmit := handlers.NewRestGenerationSubmit(generationSubmit, cfg.Logger)
	handlerGenerationGet := handlers.NewRestGenerationGet(generationGet, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(handlers.BearerChallenge)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Timeout(cfg.Rest.Timeouts.Request))
	router.Use(middleware.RequestSize(cfg.Rest.MaxRequestSize))
	router.Use(cfg.Otel.HttpHandler())
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Rest.Cors.AllowedOrigins,
		AllowedHeaders:   cfg.Rest.Cors.AllowedHeaders,
		AllowCredentials: cfg.Rest.Cors.AllowCredentials,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		},
		MaxAge: cfg.Rest.Cors.MaxAge,
	}))
	router.Use(cfg.HttpLogger.Logger())

	router.Route("/v0", func(api chi.Router) {
		api.Get("/ping", handlerPing.ServeHTTP)
		api.Get("/healthcheck", handlerHealth.ServeHTTP)

		withAuth(api, config.PermissionProjectWrite).
			Post("/projects", handlerProjectCreate.ServeHTTP)

		api.Route("/projects/{projectID}", func(project chi.Router) {
			withAuth(project, config.PermissionProjectRead).
				Get("/", handlerProjectGet.ServeHTTP)

			withAuth(project, config.PermissionProjectRead).
				Get("/ideas", handlerIdeaHistory.ServeHTTP)
			withAuth(project, config.PermissionProjectWrite).
				Post("/ideas", handlerIdeaVersionCreate.ServeHTTP)

			withAuth(project, config.PermissionProjectRead).
				Get("/step-values", handlerStepValueHistory.ServeHTTP)
			withAuth(project, config.PermissionProjectWrite).
				Post("/step-values", handlerStepValueCreate.ServeHTTP)

			withAuth(project, config.PermissionProjectRead).
				Get("/manuscripts", handlerManuscriptHistory.ServeHTTP)
			withAuth(project, config.PermissionProjectWrite).
				Post("/manuscripts", handlerManuscriptCreate.ServeHTTP)

			withAuth(project, config.PermissionGenerationWrite).
				Post("/generations", handlerGenerationSubmit.ServeHTTP)
			withAuth(project, config.PermissionGenerationRead).
				Get("/generations/{generationID}", handlerGenerationGet.ServeHTTP)
		})
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Rest.Port),
		Handler:           router,
		ReadTimeout:       cfg.Rest.Timeouts.Read,
		ReadHeaderTimeout: cfg.Rest.Timeouts.ReadHeader,
		WriteTimeout:      cfg.Rest.Timeouts.Write,
		IdleTimeout:       cfg.Rest.Timeouts.Idle,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	log.Println("Starting REST server on " + httpServer.Addr)

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down REST server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Rest.Timeouts.Request)
	defer cancel()

	err := httpServer.Shutdown(shutdownCtx)
	if err != nil {
		panic(err)
	}
}
