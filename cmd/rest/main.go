// Command rest wires the service layers together and runs the public REST server. It builds the
// DAO, core, and handler stack, mounts the routes, and serves until a termination signal triggers
// a graceful shutdown.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/env"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func main() {
	cfg := config.AppPresetDefault

	processCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx := processCtx

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

	database := lo.Must(cfg.Postgres.DB(ctx))
	defer closeDatabase(database)

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
	// TODO: Attach authorization to the Project content and generic generation routes.
	// Tracked in https://github.com/a-novel/service-narrative-engine/issues/503.
	_ = withAuth

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewPing()
	handlerHealth := handlers.NewRestHealth(jsonKeysClient, genaiClient)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewRouter(cfg.Rest, httpf.RouterOptions{
		Otel:     cfg.Otel,
		Logger:   cfg.HttpLogger,
		Liveness: "/ping",
		Methods:  []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	})

	// Probes reach /ping without a budget.
	router.Get("/ping", handlerPing.ServeHTTP)

	router.Group(func(api chi.Router) {
		api.Use(httpf.NewLimits(cfg.Rest).Standard())

		api.Get("/healthcheck", handlerHealth.ServeHTTP)
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	httpServer := httpf.NewServer(ctx, cfg.Rest, router)

	log.Println("Starting REST server on " + httpServer.Addr)

	err := httpf.Serve(ctx, httpServer, cfg.Rest.Timeouts.Shutdown)

	stop()

	if err != nil {
		panic(err)
	}
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
