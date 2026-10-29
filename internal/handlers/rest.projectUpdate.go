package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectUpdateService is the core operation [RestProjectUpdate] delegates to.
type RestProjectUpdateService interface {
	Exec(ctx context.Context, request *core.ProjectUpdateRequest) (*core.Project, error)
}

// RestProjectUpdateRequest is the JSON body the update-project endpoint accepts.
type RestProjectUpdateRequest struct {
	ID uuid.UUID `json:"id"`

	// Title names the narrative.
	Title string `json:"title"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `json:"premise"`
	// Language is the language the project is written in, one of [core.ProjectLanguages].
	Language string `json:"language"`
}

// RestProjectUpdate is the HTTP handler for the update-project endpoint. It responds with the updated
// project.
type RestProjectUpdate struct {
	service RestProjectUpdateService
	logger  logging.Log
}

func NewRestProjectUpdate(service RestProjectUpdateService, logger logging.Log) *RestProjectUpdate {
	return &RestProjectUpdate{service: service, logger: logger}
}

func (handler *RestProjectUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectUpdateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectUpdateRequest{
		OwnerID:  ownerID,
		ID:       request.ID,
		Title:    request.Title,
		Premise:  request.Premise,
		Language: request.Language,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
