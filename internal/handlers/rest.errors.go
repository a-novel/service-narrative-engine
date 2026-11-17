package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ownerErrors maps a request that names no user to its status.
var ownerErrors = httpf.ErrMap{nil: http.StatusForbidden}

// decodeErrors maps a request whose body, query or Last-Event-ID header cannot be read or parsed to its status.
var decodeErrors = httpf.ErrMap{nil: http.StatusBadRequest}

// composerErrors maps the refusals of the Scenes Composer operations to their status. A 422 answers a request
// that fails validation. A 409 answers a write that conflicts with the project's state: a full project or
// beat, a project past its byte budget, a user who owns as many projects as they may, or a delete of the
// last layer of a required kind.
var composerErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusUnprocessableEntity,
	core.ErrProjectNotFound: http.StatusNotFound,
	core.ErrBeatNotFound:    http.StatusNotFound,
	core.ErrSceneNotFound:   http.StatusNotFound,
	core.ErrLayerNotFound:   http.StatusNotFound,
	core.ErrProjectFull:     http.StatusConflict,
	core.ErrProjectBudget:   http.StatusConflict,
	core.ErrProjectsFull:    http.StatusConflict,
	core.ErrRequiredLayer:   http.StatusConflict,
}

// logoErrors maps the refusals of the logo operations to their status. A 422 answers a request that fails
// validation, and a logo the contract refuses. A refused logo carries its reason as the problem's reason tag.
var logoErrors = httpf.ErrMap{
	core.ErrInvalidRequest:      http.StatusUnprocessableEntity,
	core.ErrProjectNotFound:     http.StatusNotFound,
	core.ErrProjectLogoNotFound: http.StatusNotFound,
	picture.ErrRefused:          http.StatusUnprocessableEntity,
}

// recapErrors maps the refusals of the recap writes to their status. A 422 answers a request that fails
// validation. A 409 answers a recap that read its source at a version older than the stored recap's, or
// newer than the source's own, and one that would bring the project past its byte budget.
var recapErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusUnprocessableEntity,
	core.ErrProjectNotFound: http.StatusNotFound,
	core.ErrBeatNotFound:    http.StatusNotFound,
	core.ErrCellNotFound:    http.StatusNotFound,
	core.ErrRecapConflict:   http.StatusConflict,
	core.ErrProjectBudget:   http.StatusConflict,
}

// tagLimit tags a refusal for a storage limit with the limit it reached, so a client tells it from another
// 409: projectBytes for the bytes a project stores, projects for the projects a user owns. Every handler
// whose service keeps one of these limits passes its error through it.
func tagLimit(err error) error {
	switch {
	case errors.Is(err, core.ErrProjectBudget):
		return httpf.WithTag(err, "limit", "projectBytes")
	case errors.Is(err, core.ErrProjectsFull):
		return httpf.WithTag(err, "limit", "projects")
	default:
		return err
	}
}

// generationErrors maps the refusals of the generation operations to their status. A 422 answers a request
// that fails validation, and a generation that outgrows every call its kind allows. A 403 answers a caller
// whose roles set no caps, a 429 a generation genai admits no call of, and a 503 one genai cannot take.
var generationErrors = httpf.ErrMap{
	core.ErrInvalidRequest:     http.StatusUnprocessableEntity,
	core.ErrNoGenerationCaps:   http.StatusForbidden,
	core.ErrProjectNotFound:    http.StatusNotFound,
	core.ErrBeatNotFound:       http.StatusNotFound,
	core.ErrSceneNotFound:      http.StatusNotFound,
	core.ErrLayerNotFound:      http.StatusNotFound,
	core.ErrGenerationNotFound: http.StatusNotFound,
	core.ErrContextTooLarge:    http.StatusUnprocessableEntity,
	core.ErrGenerationQuota:    http.StatusTooManyRequests,
	core.ErrGenaiUnavailable:   http.StatusServiceUnavailable,
}

// itemFields names the invalid fields of a one-item batch request as the single-item route's body names them,
// so its invalidFields stay those of its own request: title for Beats[0].Title, id for IDs[0]. Every
// single-item body is flat, so a field's own name is its path. golib names the fields only when this tag is
// unset.
func itemFields(err error) error {
	fields, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}

	paths := make(map[string]string, len(fields))

	for _, field := range fields {
		// A field is a capitalized word, or ends in ID: Title, ProjectID, and ID itself, which IDs[0] becomes.
		name := strings.TrimSuffix(field.StructField(), "s[0]")
		if name == "ID" {
			name = "id"
		}

		paths[strings.ToLower(name[:1])+name[1:]] = field.Tag()
	}

	return httpf.WithTag(err, httpf.InvalidFieldsTag, paths)
}
