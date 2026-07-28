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
	// TODO: Attach authorization to the Idea and generation routes.
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

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
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

	router.Get("/ping", handlerPing.ServeHTTP)
	router.Get("/healthcheck", handlerHealth.ServeHTTP)

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
