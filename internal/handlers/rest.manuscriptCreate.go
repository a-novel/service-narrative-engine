package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// RestManuscriptCreateService saves one complete Manuscript Version.
type RestManuscriptCreateService interface {
	Exec(ctx context.Context, request *core.ManuscriptCreateRequest) (*core.Manuscript, error)
}

// RestManuscriptCreateRequest contains one complete static Manuscript value.
type RestManuscriptCreateRequest struct {
	Value json.RawMessage `json:"value"`
}

// RestManuscriptCreate publishes Manuscript version writes over HTTP.
type RestManuscriptCreate struct {
	service RestManuscriptCreateService
	logger  logging.Log
}

// NewRestManuscriptCreate creates a Manuscript Version handler.
func NewRestManuscriptCreate(
	service RestManuscriptCreateService,
	logger logging.Log,
) *RestManuscriptCreate {
	return &RestManuscriptCreate{service: service, logger: logger}
}

// ServeHTTP appends one Manuscript Version under an owned Project.
func (handler *RestManuscriptCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ManuscriptCreate")
	defer span.End()

	actor, projectID, err := restProjectIdentity(r)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	var request RestManuscriptCreateRequest

	err = decodeRestJSON(r, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restTransportErrors, err)

		return
	}

	manuscript, err := handler.service.Exec(ctx, &core.ManuscriptCreateRequest{
		Actor:      *actor,
		ProjectID:  projectID,
		Manuscript: request.Value,
	})
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, restServiceErrors, err)

		return
	}

	httpf.SendJSONStatus(ctx, w, span, http.StatusCreated, loadRestManuscriptVersion(manuscript))
}
