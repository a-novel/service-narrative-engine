package handlers

import (
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ownerErrors maps a request that names no user to its status.
var ownerErrors = httpf.ErrMap{nil: http.StatusForbidden}

// composerErrors maps the refusals of the Scenes Composer operations to their status. A 409 answers a write
// that conflicts with the project's state: a full project or beat, a project past its byte budget, a user
// who owns as many projects as they may, or a delete of the last layer of a required kind.
var composerErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusBadRequest,
	core.ErrProjectNotFound: http.StatusNotFound,
	core.ErrBeatNotFound:    http.StatusNotFound,
	core.ErrSceneNotFound:   http.StatusNotFound,
	core.ErrLayerNotFound:   http.StatusNotFound,
	core.ErrProjectFull:     http.StatusConflict,
	core.ErrProjectBudget:   http.StatusConflict,
	core.ErrProjectsFull:    http.StatusConflict,
	core.ErrRequiredLayer:   http.StatusConflict,
}

// recapErrors maps the refusals of the recap writes to their status. A 409 answers a recap that read its
// source at a version older than the stored recap's, or newer than the source's own, and one that would
// bring the project past its byte budget.
var recapErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusBadRequest,
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

// generationErrors maps the refusals of the generation operations to their status. A 403 answers a caller
// whose roles set no caps, a 422 a generation that outgrows every call its kind allows, a 429 one genai
// admits no call of, and a 503 one genai cannot take.
var generationErrors = httpf.ErrMap{
	core.ErrInvalidRequest:     http.StatusBadRequest,
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
