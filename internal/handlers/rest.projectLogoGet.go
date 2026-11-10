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

// RestProjectLogoGetService is the core operation [RestProjectLogoGet] delegates to.
type RestProjectLogoGetService interface {
	Exec(ctx context.Context, request *core.ProjectLogoGetRequest) (*core.ProjectLogo, error)
}

// RestProjectLogoGetRequest is the query parameters the get-logo endpoint accepts.
type RestProjectLogoGetRequest struct {
	ProjectID uuid.UUID `schema:"projectID"`
}

// RestProjectLogoGet is the HTTP handler for the get-logo endpoint. It responds with the bytes of the logo
// of a project of the caller.
//
// The logo carries the media type it was stored with, which X-Content-Type-Options keeps the browser from
// second-guessing. Its strong ETag (RFC 9110 section 8.8.3) and Last-Modified come from its version, so
// [http.ServeContent] answers a matching conditional request with 304. Cache-Control is no-cache (RFC 9111
// section 5.2.2.4): a cache revalidates the logo on every use. The platform caches for good under a URL
// that names the version.
type RestProjectLogoGet struct {
	service RestProjectLogoGetService
	logger  logging.Log
}

func NewRestProjectLogoGet(service RestProjectLogoGetService, logger logging.Log) *RestProjectLogoGet {
	return &RestProjectLogoGet{service: service, logger: logger}
}

func (handler *RestProjectLogoGet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectLogoGet")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectLogoGetRequest

	err = muxDecoder.Decode(&request, r.URL.Query())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	result, err := handler.service.Exec(ctx, &core.ProjectLogoGetRequest{OwnerID: ownerID, ProjectID: request.ProjectID})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, logoErrors, err)

		return
	}

	header := w.Header()
	header.Set("Content-Type", result.MediaType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cache-Control", "no-cache")
	header.Set("ETag", `"`+strconv.FormatInt(result.Version.UnixMicro(), 10)+`"`)

	http.ServeContent(w, r, "", result.Version, bytes.NewReader(result.Data))
}
