package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectEditsListService is the core operation [RestProjectEditsList] delegates to.
type RestProjectEditsListService interface {
	Exec(ctx context.Context, request *core.ProjectEditsListRequest) (*core.ProjectEdits, error)
}

// RestProjectEdits is the REST representation of when each project of the caller last changed.
type RestProjectEdits struct {
	// Edits maps the ID of each project of the caller to its editedAt.
	Edits map[uuid.UUID]time.Time `json:"edits"`
}

// RestProjectEditsList is the HTTP handler for the list-edits endpoint. It responds with the editedAt of every
// project of the caller.
type RestProjectEditsList struct {
	service RestProjectEditsListService
	logger  logging.Log
}

func NewRestProjectEditsList(service RestProjectEditsListService, logger logging.Log) *RestProjectEditsList {
	return &RestProjectEditsList{service: service, logger: logger}
}

func (handler *RestProjectEditsList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectEditsList")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectEditsListRequest{OwnerID: ownerID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, RestProjectEdits{Edits: result.Edits})
}
