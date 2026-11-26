package handlers

import (
	"context"
	"net/http"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestProjectCreateService is the core operation [RestProjectCreate] delegates to.
type RestProjectCreateService interface {
	Exec(ctx context.Context, request *core.ProjectCreateRequest) (*core.Project, error)
}

// RestProjectCreateRequest is the JSON body the create-project endpoint accepts.
type RestProjectCreateRequest struct {
	// Title names the narrative.
	Title string `json:"title"`
	// Premise says what the narrative is about, in a few sentences.
	Premise string `json:"premise"`
	// Language is the language the project is written in, one of [core.ProjectLanguages].
	Language string `json:"language"`
	// Tags are the author's custom labels, in order, none when omitted.
	Tags []string `json:"tags"`
	// Genres are the genres the author names, in order, none when omitted.
	Genres []string `json:"genres"`
}

// RestProjectCreate is the HTTP handler for the create-project endpoint. It responds 201 with the
// project, and a Location naming the project's read (RFC 9110 §15.3.2).
type RestProjectCreate struct {
	service RestProjectCreateService
	logger  logging.Log
}

func NewRestProjectCreate(service RestProjectCreateService, logger logging.Log) *RestProjectCreate {
	return &RestProjectCreate{service: service, logger: logger}
}

func (handler *RestProjectCreate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.ProjectCreate")
	defer span.End()

	ownerID, err := serviceauthentication.RequestUser(r.Context())
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, ownerErrors, err)

		return
	}

	var request RestProjectCreateRequest

	err = httpf.DecodeJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	serviceRequest := &core.ProjectCreateRequest{
		OwnerID:  ownerID,
		Title:    request.Title,
		Premise:  request.Premise,
		Language: request.Language,
		Tags:     request.Tags,
		Genres:   request.Genres,
	}

	result, err := handler.service.Exec(ctx, serviceRequest)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, projectErrors, tagLimit(err))

		return
	}

	w.Header().Set("Location", "/v1/project?id="+result.ID.String())
	httpf.SendJSONStatus(w, span, http.StatusCreated, loadProject(result))
}
