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

// RestProjectLogosDriftService is the core operation [RestProjectLogosDrift] delegates to.
type RestProjectLogosDriftService interface {
	Exec(ctx context.Context, request *core.ProjectLogosDriftRequest) (*core.ProjectLogosDriftResult, error)
}

// RestProjectLogosDriftRequest is the JSON body the logos drift endpoint accepts.
type RestProjectLogosDriftRequest struct {
	// Logos maps the ID of each project whose logo the client holds to the logo's logoChecksum.
	Logos map[string]string `json:"logos"`
}

// RestProjectLogosDriftResult is the REST representation of the drift between the logos a client holds and those
// of the caller's projects.
type RestProjectLogosDriftResult struct {
	// Logos maps each project of the caller whose logo the request leaves out, or names with another checksum, to
	// the logo's checksum.
	Logos map[uuid.UUID]string `json:"logos"`
	// Removed are the IDs of the request whose project is not the caller's or holds no logo.
	Removed []uuid.UUID `json:"removed"`
}

// RestProjectLogosDrift is the HTTP handler for the logos drift endpoint. It responds with the checksums of the
// caller's logos that drifted from those the client holds, and the held IDs whose project holds no logo of the
// caller.
type RestProjectLogosDrift struct {
	service RestProjectLogosDriftService
	logger  logging.Log
}

func NewRestProjectLogosDrift(service RestProjectLogosDriftService, logger logging.Log) *RestProjectLogosDrift {
	return &RestProjectLogosDrift{service: service, logger: logger}
}

func (handler *RestProjectLogosDrift) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectLogosDrift")
	defer span.End()

	ownerID, err := requestOwner(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectLogosDriftRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, httpf.ErrMap{nil: http.StatusBadRequest}, err)

		return
	}

	serviceRequest := &core.ProjectLogosDriftRequest{
		OwnerID: ownerID,
		Logos:   request.Logos,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, composerErrors, err)

		return
	}

	httpf.SendJSONStatus(w, span, http.StatusOK, RestProjectLogosDriftResult{Logos: result.Logos, Removed: result.Removed})
}
