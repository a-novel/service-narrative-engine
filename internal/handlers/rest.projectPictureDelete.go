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

// RestProjectPictureDeleteService is the core operation [RestProjectPictureDelete] delegates to.
type RestProjectPictureDeleteService interface {
	Exec(ctx context.Context, request *core.ProjectPictureDeleteRequest) (*core.Project, error)
}

// RestProjectPictureDeleteRequest is the query parameters the delete-picture endpoint accepts.
type RestProjectPictureDeleteRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// Picture names the picture: logo or background.
	Picture string `schema:"picture"`
}

// RestProjectPictureDelete is the HTTP handler for the delete-picture endpoint. It responds with the
// project, without the picture.
type RestProjectPictureDelete struct {
	service RestProjectPictureDeleteService
	logger  logging.Log
}

func NewRestProjectPictureDelete(
	service RestProjectPictureDeleteService, logger logging.Log,
) *RestProjectPictureDelete {
	return &RestProjectPictureDelete{service: service, logger: logger}
}

func (handler *RestProjectPictureDelete) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectPictureDelete")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectPictureDeleteRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectPictureDeleteRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Picture:   request.Picture,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pictureErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
