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

// RestProjectPictureUpdateService is the core operation [RestProjectPictureUpdate] delegates to.
type RestProjectPictureUpdateService interface {
	Exec(ctx context.Context, request *core.ProjectPictureUpdateRequest) (*core.Project, error)
}

// RestProjectPictureUpdateRequest is the query parameters the update-picture endpoint accepts.
type RestProjectPictureUpdateRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// Picture names the picture: logo or background.
	Picture string `schema:"picture"`
}

// RestProjectPictureUpdate is the HTTP handler for the update-picture endpoint. Its body is the encoded
// image, whose bytes name its format whatever the Content-Type says. It responds with the project.
type RestProjectPictureUpdate struct {
	service RestProjectPictureUpdateService
	logger  logging.Log
}

func NewRestProjectPictureUpdate(
	service RestProjectPictureUpdateService, logger logging.Log,
) *RestProjectPictureUpdate {
	return &RestProjectPictureUpdate{service: service, logger: logger}
}

func (handler *RestProjectPictureUpdate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectPictureUpdate")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectPictureUpdateRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	// The picture is decoded whole, so the body is read whole, within the router's limit.
	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectPictureUpdateRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Picture:   request.Picture,
		Data:      data,
	})

	var refused picture.RefusedError
	if errors.As(err, &refused) {
		err = httpf.WithTag(err, "reason", string(refused))
	}

	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pictureErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, loadProject(result))
}
