package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/samber/lo"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestInstancesCreateService is the core operation [RestInstancesCreate] delegates to.
type RestInstancesCreateService interface {
	Exec(ctx context.Context, request *core.InstancesCreateRequest) ([]*core.InstanceDescriptor, error)
}

// RestInstancesCreateItem is one instance of a [RestInstancesCreateRequest].
type RestInstancesCreateItem struct {
	// Plugin is the name of the plugin, as the registry holds it.
	Plugin string `json:"plugin"`
	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string `json:"version"`
	// Name is the display name the writer gives the instance.
	Name string `json:"name"`

	// Content is the JSON document the plugin edits.
	Content json.RawMessage `json:"content"`
}

// RestInstancesCreateRequest is the JSON body the create-instances endpoint accepts.
type RestInstancesCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Instances lists the instances to create.
	Instances []RestInstancesCreateItem `json:"instances"`
}

// RestInstancesCreate is the HTTP handler for the create-instances endpoint. It responds 201 with the new
// instances in the order of the request, without their content, and a Location naming the list of the
// project's instances.
type RestInstancesCreate struct {
	service RestInstancesCreateService
	logger  logging.Log
}

func NewRestInstancesCreate(service RestInstancesCreateService, logger logging.Log) *RestInstancesCreate {
	return &RestInstancesCreate{service: service, logger: logger}
}

func (handler *RestInstancesCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstancesCreate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstancesCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstancesCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Instances: lo.Map(request.Instances, func(item RestInstancesCreateItem, _ int) core.InstancesCreateItem {
			return core.InstancesCreateItem(item)
		}),
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, tagLimit(tagField(err)))

		return
	}

	response := lo.Map(result, func(instance *core.InstanceDescriptor, _ int) InstanceDescriptor {
		return loadInstanceDescriptor(instance)
	})

	w.Header().Set("Location", "/v1/instances?projectID="+request.ProjectID.String())
	httpf.SendJSONStatus(w, span, http.StatusCreated, response)
}
