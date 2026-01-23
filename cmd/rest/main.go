package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/samber/lo"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	"github.com/a-novel/service-narrative-engine/internal/services"
)

func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	jsonKeysCredentials := lo.Must(cfg.DependenciesConfig.ServiceJsonKeysCredentials.Options(ctx))

	jsonKeysClient := lo.Must(jkpkg.NewClient(
		fmt.Sprintf("%s:%d", cfg.DependenciesConfig.ServiceJsonKeysHost, cfg.DependenciesConfig.ServiceJsonKeysPort),
		jsonKeysCredentials...,
	))

	serviceVerifyAccessToken := jkpkg.NewClaimsVerifier[authpkg.Claims](jsonKeysClient)

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	repositoryModuleInsert := dao.NewModuleInsert()
	repositoryModuleSelect := dao.NewModuleSelect()
	repositoryModuleDelete := dao.NewModuleDelete()
	repositoryModuleListVersions := dao.NewModuleListVersions()
	repositoryModuleGenerate := dao.NewModuleGenerate()

	repositoryProjectInsert := dao.NewProjectInsert()
	repositoryProjectSelect := dao.NewProjectSelect()
	repositoryProjectDelete := dao.NewProjectDelete()
	repositoryProjectList := dao.NewProjectList()
	repositoryProjectUpdate := dao.NewProjectUpdate()

	repositorySchemaInsert := dao.NewSchemaInsert()
	repositorySchemaSelect := dao.NewSchemaGet()
	repositorySchemaUpdate := dao.NewSchemaUpdate()
	repositorySchemaList := dao.NewSchemaList()
	repositorySchemaListVersions := dao.NewSchemaListVersions()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceModuleCreate := services.NewModuleCreate(repositoryModuleInsert, repositoryModuleDelete)
	serviceModuleSelect := services.NewModuleSelect(repositoryModuleSelect)
	serviceModuleListVersions := services.NewModuleListVersions(repositoryModuleListVersions)

	serviceProjectInit := services.NewProjectInit(repositoryProjectInsert, repositorySchemaInsert, repositoryModuleSelect)
	serviceProjectDelete := services.NewProjectDelete(repositoryProjectDelete, repositoryProjectSelect)
	serviceProjectList := services.NewProjectList(repositoryProjectList)
	serviceProjectUpdate := services.NewProjectUpdate(
		repositoryProjectUpdate,
		repositoryProjectSelect,
		repositorySchemaInsert,
		repositoryModuleSelect,
	)

	serviceSchemaCreate := services.NewSchemaCreate(
		repositorySchemaInsert,
		repositoryProjectSelect,
		repositoryModuleSelect,
	)
	serviceSchemaGenerate := services.NewSchemaGenerate(
		repositoryModuleGenerate,
		repositorySchemaList,
		repositorySchemaInsert,
		repositoryProjectSelect,
		repositoryModuleSelect,
	)
	serviceSchemaSelect := services.NewSchemaSelect(repositorySchemaSelect, repositoryProjectSelect)
	serviceSchemaRewrite := services.NewSchemaRewrite(
		repositorySchemaUpdate,
		repositoryProjectSelect,
		repositorySchemaSelect,
	)
	serviceSchemaListVersions := services.NewSchemaListVersions(repositorySchemaListVersions, repositoryProjectSelect)

	// Unused for now, but available for system module loading
	_ = serviceModuleCreate

	// =================================================================================================================
	// MIDDLEWARES
	// =================================================================================================================

	withAuth := authpkg.NewAuthHandler(serviceVerifyAccessToken, cfg.Permissions)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewPing()
	handlerHealth := handlers.NewHealth(jsonKeysClient)

	handlerModuleSelect := handlers.NewModuleSelect(serviceModuleSelect)
	handlerModuleListVersions := handlers.NewModuleListVersions(serviceModuleListVersions)

	handlerProjectInit := handlers.NewProjectInit(serviceProjectInit)
	handlerProjectDelete := handlers.NewProjectDelete(serviceProjectDelete)
	handlerProjectList := handlers.NewProjectList(serviceProjectList)
	handlerProjectUpdate := handlers.NewProjectUpdate(serviceProjectUpdate)

	handlerSchemaCreate := handlers.NewSchemaCreate(serviceSchemaCreate)
	handlerSchemaGenerate := handlers.NewSchemaGenerate(serviceSchemaGenerate)
	handlerSchemaSelect := handlers.NewSchemaSelect(serviceSchemaSelect)
	handlerSchemaRewrite := handlers.NewSchemaRewrite(serviceSchemaRewrite)
	handlerSchemaListVersions := handlers.NewSchemaListVersions(serviceSchemaListVersions)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(cfg.Api.Timeouts.Request))
	router.Use(middleware.RequestSize(cfg.Api.MaxRequestSize))
	router.Use(cfg.Otel.HttpHandler())
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Api.Cors.AllowedOrigins,
		AllowedHeaders:   cfg.Api.Cors.AllowedHeaders,
		AllowCredentials: cfg.Api.Cors.AllowCredentials,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		MaxAge: cfg.Api.Cors.MaxAge,
	}))
	router.Use(cfg.Logger.Logger())

	router.Get("/ping", handlerPing.ServeHTTP)
	router.Get("/healthcheck", handlerHealth.ServeHTTP)

	router.Route("/modules", func(r chi.Router) {
		withAuth(r, "modules:get").Get("/", handlerModuleSelect.ServeHTTP)
		withAuth(r, "modules:versions:list").Get("/versions", handlerModuleListVersions.ServeHTTP)
	})

	router.Route("/projects", func(r chi.Router) {
		withAuth(r, "projects:list").Get("/", handlerProjectList.ServeHTTP)
		withAuth(r, "projects:create").Put("/", handlerProjectInit.ServeHTTP)
		withAuth(r, "projects:update").Patch("/", handlerProjectUpdate.ServeHTTP)
		withAuth(r, "projects:delete").Delete("/", handlerProjectDelete.ServeHTTP)
	})

	router.Route("/schemas", func(r chi.Router) {
		withAuth(r, "schemas:get").Get("/", handlerSchemaSelect.ServeHTTP)
		withAuth(r, "schemas:versions:list").Get("/versions", handlerSchemaListVersions.ServeHTTP)
		withAuth(r, "schemas:create").Put("/", handlerSchemaCreate.ServeHTTP)
		withAuth(r, "schemas:generate").Put("/generate", handlerSchemaGenerate.ServeHTTP)
		withAuth(r, "schemas:rewrite").Patch("/", handlerSchemaRewrite.ServeHTTP)
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Api.Port),
		Handler:           router,
		ReadTimeout:       cfg.Api.Timeouts.Read,
		ReadHeaderTimeout: cfg.Api.Timeouts.ReadHeader,
		WriteTimeout:      cfg.Api.Timeouts.Write,
		IdleTimeout:       cfg.Api.Timeouts.Idle,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	log.Println("Starting server on " + httpServer.Addr)

	err := httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err.Error())
	}
}
