package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestInstanceCreateService is the batch operation [RestInstanceCreate] delegates to, with a list of one.
type RestInstanceCreateService interface {
	Exec(ctx context.Context, request *core.InstancesCreateRequest) ([]*core.InstanceDescriptor, error)
}

// RestInstanceCreateRequest is the JSON body the create-instance endpoint accepts.
type RestInstanceCreateRequest struct {
	ProjectID uuid.UUID `json:"projectID"`

	// Plugin is the name of the plugin, as the registry holds it.
	Plugin string `json:"plugin"`
	// Version is the exact version of the plugin the content follows, as a semantic version.
	Version string `json:"version"`
	// Name is the display name the writer gives the instance.
	Name string `json:"name"`

	// Content is the JSON document the plugin edits.
	Content json.RawMessage `json:"content"`
}

// RestInstanceCreate is the HTTP handler for the create-instance endpoint. It responds 201 with the instance
// without its content, and a Location naming the instance's read (RFC 9110 §15.3.2).
type RestInstanceCreate struct {
	service RestInstanceCreateService
	logger  logging.Log
}

func NewRestInstanceCreate(service RestInstanceCreateService, logger logging.Log) *RestInstanceCreate {
	return &RestInstanceCreate{service: service, logger: logger}
}

func (handler *RestInstanceCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.InstanceCreate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestInstanceCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.InstancesCreateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Instances: []core.InstancesCreateItem{{
			Plugin:  request.Plugin,
			Version: request.Version,
			Name:    request.Name,
			Content: request.Content,
		}},
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, instanceErrors, tagLimit(itemFields(err)))

		return
	}

	w.Header().Set("Location", instanceLocation(result[0]))
	httpf.SendJSONStatus(w, span, http.StatusCreated, loadInstanceDescriptor(result[0]))
}
