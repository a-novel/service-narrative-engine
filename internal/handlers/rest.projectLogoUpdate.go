package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectLogoUpdateService is the core operation [RestProjectLogoUpdate] delegates to.
type RestProjectLogoUpdateService interface {
	Exec(ctx context.Context, request *core.ProjectLogoUpdateRequest) (*core.Project, error)
}

// RestProjectLogoUpdateRequest is the query parameters the update-logo endpoint accepts.
type RestProjectLogoUpdateRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
}

// RestProjectLogoUpdate is the HTTP handler for the update-logo endpoint. Its body is the encoded
// image, whose bytes name its format whatever the Content-Type says. It responds with the project.
type RestProjectLogoUpdate struct {
	service RestProjectLogoUpdateService
	logger  logging.Log
}

func NewRestProjectLogoUpdate(
	service RestProjectLogoUpdateService, logger logging.Log,
) *RestProjectLogoUpdate {
	return &RestProjectLogoUpdate{service: service, logger: logger}
}

func (handler *RestProjectLogoUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectLogoUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectLogoUpdateRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	// The logo is decoded whole, so the body is read whole, within the router's limit.
	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectLogoUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Data:      data,
	})

	var refused picture.RefusedError
	if errors.As(err, &refused) {
		err = httpf.WithTag(err, "reason", string(refused))
	}

	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, logoErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
