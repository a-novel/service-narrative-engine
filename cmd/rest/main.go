package main

import (
	"context"
	"errors"
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

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-template/internal/config"
	"github.com/a-novel/service-template/internal/config/env"
	"github.com/a-novel/service-template/internal/dao"
	"github.com/a-novel/service-template/internal/handlers"
	"github.com/a-novel/service-template/internal/services"
)

// Runs the main REST server.
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

	handlerPing := handlers.NewPing()
	handlerHealth := handlers.NewRestHealth()
	handlerItemCreate := handlers.NewItemCreatePublic(serviceItemCreate, cfg.Log)
	handlerItemGet := handlers.NewItemGetPublic(serviceItemGet, cfg.Log)
	handlerItemList := handlers.NewItemListPublic(serviceItemList, cfg.Log)
	handlerItemUpdate := handlers.NewItemUpdatePublic(serviceItemUpdate, cfg.Log)
	handlerItemDelete := handlers.NewItemDeletePublic(serviceItemDelete, cfg.Log)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
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
	router.Post("/items", handlerItemCreate.ServeHTTP)
	router.Get("/items", handlerItemList.ServeHTTP)
	router.Get("/item", handlerItemGet.ServeHTTP)
	router.Put("/item", handlerItemUpdate.ServeHTTP)
	router.Delete("/item", handlerItemDelete.ServeHTTP)

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
