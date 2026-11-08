package handlers

import (
	"bytes"
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestProjectPictureGetService is the core operation [RestProjectPictureGet] delegates to.
type RestProjectPictureGetService interface {
	Exec(ctx context.Context, request *core.ProjectPictureGetRequest) (*core.ProjectPicture, error)
}

// RestProjectPictureGetRequest is the query parameters the get-picture endpoint accepts.
type RestProjectPictureGetRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
	// Picture names the picture: logo or background.
	Picture string `schema:"picture"`
}

// RestProjectPictureGet is the HTTP handler for the get-picture endpoint. It responds with the bytes of a
// picture of a project of the caller.
//
// The picture carries the media type it was stored with, which X-Content-Type-Options keeps the browser
// from second-guessing. Its strong ETag (RFC 9110 section 8.8.3) and Last-Modified come from its version,
// so [http.ServeContent] answers a matching conditional request with 304. Cache-Control is no-cache (RFC
// 9111 section 5.2.2.4): a cache revalidates the picture on every use. The platform caches for good under
// a URL that names the version.
type RestProjectPictureGet struct {
	service RestProjectPictureGetService
	logger  logging.Log
}

func NewRestProjectPictureGet(service RestProjectPictureGetService, logger logging.Log) *RestProjectPictureGet {
	return &RestProjectPictureGet{service: service, logger: logger}
}

func (handler *RestProjectPictureGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectPictureGet")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectPictureGetRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectPictureGetRequest{
		OwnerID:   ownerID,
		ProjectID: request.ProjectID,
		Picture:   request.Picture,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pictureErrors, err)

		return
	}

	header := w.Header()
	header.Set("Content-Type", result.MediaType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "no-cache")
	header.Set("ETag", `"`+strconv.FormatInt(result.Version.UnixMicro(), 10)+`"`)

	http.ServeContent(w, r, "", result.Version, bytes.NewReader(result.Data))
}
