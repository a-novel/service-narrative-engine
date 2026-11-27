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

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/handlers"
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

	router := lo.Must(newRouter(cfg, jsonKeysClient))

	log.Println("Starting REST server on :" + strconv.Itoa(cfg.Rest.Port))

	lo.Must0(httpf.ListenAndServe(ctx, cfg.Rest, router))
}

// newRouter builds the API behind the service-token gate, which wraps the whole router: a request
// reaches a route only with a token cfg accepts, and the mux reads the client IP header and counts
// rate limits for verified callers alone. The liveness route stays open to the kubelet. The gate
// refuses an empty issuer unless cfg opts a local stack out, so the server never starts open by
// mistake.
func newRouter(cfg config.App, jsonKeysClient servicejsonkeys.Client) (http.Handler, error) {
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
	daoProjectDefaultUpdate := dao.NewPgProjectDefaultUpdate()
	daoProjectLinksSelect := dao.NewPgProjectLinksSelect()
	daoProjectLinksUpsert := dao.NewPgProjectLinksUpsert()
	daoInstanceCount := dao.NewPgInstanceCount()
	daoInstancesInsert := dao.NewPgInstancesInsert()
	daoInstanceList := dao.NewPgInstanceList()
	daoInstanceSelect := dao.NewPgInstanceSelect()
	daoInstanceUpdate := dao.NewPgInstanceUpdate()
	daoInstanceContentUpdate := dao.NewPgInstanceContentUpdate()
	daoInstanceDelete := dao.NewPgInstanceDelete()
	daoPluginList := dao.NewPgPluginList()
	daoPluginManifestSelect := dao.NewPgPluginManifestSelect()
	daoPluginModuleSelect := dao.NewPgPluginModuleSelect()
	daoPluginMessagesSelect := dao.NewPgPluginMessagesSelect()
	daoPluginVersionList := dao.NewPgPluginVersionList()
	transactor := postgres.NewTransactor(nil)

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceProjectCreate := core.NewProjectCreate(daoOwnerLock, daoProjectCount, daoProjectInsert, transactor)
	serviceProjectGet := core.NewProjectGet(daoProjectOwned, daoProjectSelect)
	serviceProjectFieldsDrift := core.NewProjectFieldsDrift(daoProjectFieldsDrift)
	serviceProjectEditsList := core.NewProjectEditsList(daoProjectEditsList)
	serviceProjectLogosDrift := core.NewProjectLogosDrift(daoProjectLogosDrift)
	serviceProjectUpdate := core.NewProjectUpdate(daoProjectLock, daoProjectUpdate, daoProjectBytes, transactor)
	serviceProjectDelete := core.NewProjectDelete(daoProjectOwned, daoProjectDelete)
	serviceProjectLogoUpdate := core.NewProjectLogoUpdate(daoProjectOwned, daoProjectLogoUpsert)
	serviceProjectLogoGet := core.NewProjectLogoGet(daoProjectOwned, daoProjectLogoSelect)
	serviceProjectLogoDelete := core.NewProjectLogoDelete(daoProjectOwned, daoProjectLogoDelete)
	serviceProjectDefaultUpdate := core.NewProjectDefaultUpdate(daoProjectLock, daoProjectDefaultUpdate, transactor)
	serviceProjectLinksGet := core.NewProjectLinksGet(daoProjectOwned, daoProjectLinksSelect)
	serviceProjectLinksUpdate := core.NewProjectLinksUpdate(
		daoProjectLock, daoProjectLinksUpsert, daoProjectBytes, transactor, cfg.Caps,
	)
	serviceInstancesCreate := core.NewInstancesCreate(
		daoProjectLock, daoInstanceCount, daoInstancesInsert, daoProjectDefaultUpdate, daoProjectBytes, transactor,
		cfg.Caps,
	)
	serviceInstanceList := core.NewInstanceList(daoProjectOwned, daoInstanceList)
	serviceInstanceGet := core.NewInstanceGet(daoProjectOwned, daoInstanceSelect)
	serviceInstanceUpdate := core.NewInstanceUpdate(daoProjectLock, daoInstanceUpdate, daoProjectBytes, transactor)
	serviceInstanceContentUpdate := core.NewInstanceContentUpdate(
		daoProjectLock, daoInstanceContentUpdate, daoProjectBytes, transactor, cfg.Caps,
	)
	serviceInstanceDelete := core.NewInstanceDelete(daoProjectLock, daoInstanceDelete, transactor)
	servicePluginList := core.NewPluginList(daoPluginList)
	servicePluginManifestGet := core.NewPluginManifestGet(daoPluginManifestSelect)
	servicePluginModuleGet := core.NewPluginModuleGet(daoPluginModuleSelect)
	servicePluginMessagesGet := core.NewPluginMessagesGet(daoPluginMessagesSelect)
	servicePluginVersionList := core.NewPluginVersionList(daoPluginVersionList)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerHealth := handlers.NewRestHealth(jsonKeysClient)
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
	handlerProjectDefaultUpdate := handlers.NewRestProjectDefaultUpdate(serviceProjectDefaultUpdate, cfg.Logger)
	handlerProjectLinksGet := handlers.NewRestProjectLinksGet(serviceProjectLinksGet, cfg.Logger)
	handlerProjectLinksUpdate := handlers.NewRestProjectLinksUpdate(serviceProjectLinksUpdate, cfg.Logger)
	handlerInstanceCreate := handlers.NewRestInstanceCreate(serviceInstancesCreate, cfg.Logger)
	handlerInstancesCreate := handlers.NewRestInstancesCreate(serviceInstancesCreate, cfg.Logger)
	handlerInstanceList := handlers.NewRestInstanceList(serviceInstanceList, cfg.Logger)
	handlerInstanceGet := handlers.NewRestInstanceGet(serviceInstanceGet, cfg.Logger)
	handlerInstanceUpdate := handlers.NewRestInstanceUpdate(serviceInstanceUpdate, cfg.Logger)
	handlerInstanceContentUpdate := handlers.NewRestInstanceContentUpdate(serviceInstanceContentUpdate, cfg.Logger)
	handlerInstanceDelete := handlers.NewRestInstanceDelete(serviceInstanceDelete, cfg.Logger)
	handlerPluginList := handlers.NewRestPluginList(servicePluginList, cfg.Logger)
	handlerPluginManifestGet := handlers.NewRestPluginManifestGet(servicePluginManifestGet, cfg.Logger)
	handlerPluginUIGet := handlers.NewRestPluginModuleGet(servicePluginModuleGet, core.PluginModuleUI, cfg.Logger)
	handlerPluginMigrationGet := handlers.NewRestPluginModuleGet(
		servicePluginModuleGet, core.PluginModuleMigration, cfg.Logger,
	)
	handlerPluginMessagesGet := handlers.NewRestPluginMessagesGet(servicePluginMessagesGet, cfg.Logger)
	handlerPluginVersionList := handlers.NewRestPluginVersionList(servicePluginVersionList, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewMux(cfg.Rest, cfg.RestLogger, cfg.App.DowntimeStart)
	limit := handlers.NewRestLimits(cfg.Rest, cfg.Limits)

	router.Route("/v1", func(api chi.Router) {
		// The router's body limit fits a batch; a route that writes one item, or compares what a client holds,
		// caps its body lower.
		item := api.With(middleware.RequestSize(cfg.Limits.MaxItemRequestSize))

		api.Get("/ping", httpf.Ping)
		api.Get("/healthcheck", handlerHealth)

		withAuth(item, config.PermissionProjectWrite).Post("/projects", handlerProjectCreate.ServeHTTP)
		withAuth(item, config.PermissionProjectRead).Post("/projects/fields/drift", handlerProjectFieldsDrift.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/projects/edits", handlerProjectEditsList.ServeHTTP)
		withAuth(item, config.PermissionProjectRead).Post("/projects/logos/drift", handlerProjectLogosDrift.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/project", handlerProjectGet.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/project", handlerProjectUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/project", handlerProjectDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/project/logo", handlerProjectLogoUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/project/logo", handlerProjectLogoGet.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/project/logo", handlerProjectLogoDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/project/default", handlerProjectDefaultUpdate.ServeHTTP)

		// A route that writes a document, a content or the links, also counts the writes of each signed-in user.
		withAuth(api, config.PermissionProjectRead).Get("/instances", handlerInstanceList.ServeHTTP)
		withAuth(api, config.PermissionProjectWrite).With(limit.DocumentWrite).
			Post("/instances", handlerInstancesCreate.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/instance", handlerInstanceGet.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).With(limit.DocumentWrite).
			Post("/instance", handlerInstanceCreate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Put("/instance", handlerInstanceUpdate.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).Delete("/instance", handlerInstanceDelete.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).With(limit.DocumentWrite).
			Put("/instance/content", handlerInstanceContentUpdate.ServeHTTP)
		withAuth(api, config.PermissionProjectRead).Get("/links", handlerProjectLinksGet.ServeHTTP)
		withAuth(item, config.PermissionProjectWrite).With(limit.DocumentWrite).
			Put("/links", handlerProjectLinksUpdate.ServeHTTP)

		withAuth(api, config.PermissionPluginRead).Get("/plugins", handlerPluginList.ServeHTTP)
		withAuth(api, config.PermissionPluginRead).Get("/plugin/version", handlerPluginManifestGet.ServeHTTP)
		withAuth(api, config.PermissionPluginRead).Get("/plugin/version/ui", handlerPluginUIGet.ServeHTTP)
		withAuth(api, config.PermissionPluginRead).Get("/plugin/version/migration", handlerPluginMigrationGet.ServeHTTP)
		withAuth(api, config.PermissionPluginRead).Get("/plugin/version/messages", handlerPluginMessagesGet.ServeHTTP)
		withAuth(api, config.PermissionPluginRead).Get("/plugin/versions", handlerPluginVersionList.ServeHTTP)
	})

	return withServiceToken(router), nil
}
