package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ownerErrors maps a request that names no user to its status.
var ownerErrors = httpf.ErrMap{nil: http.StatusForbidden}

// composerErrors maps the refusals of the Scenes Composer operations to their status. A 422 answers a
// full project or beat, or a delete of the last layer of a required kind.
var composerErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusBadRequest,
	core.ErrProjectNotFound: http.StatusNotFound,
	core.ErrBeatNotFound:    http.StatusNotFound,
	core.ErrSceneNotFound:   http.StatusNotFound,
	core.ErrLayerNotFound:   http.StatusNotFound,
	core.ErrProjectFull:     http.StatusUnprocessableEntity,
	core.ErrRequiredLayer:   http.StatusUnprocessableEntity,
}

// recapErrors maps the refusals of the recap writes to their status. A 409 answers a recap that read its
// source at a version older than the stored recap's, or newer than the source's own.
var recapErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusBadRequest,
	core.ErrProjectNotFound: http.StatusNotFound,
	core.ErrBeatNotFound:    http.StatusNotFound,
	core.ErrCellNotFound:    http.StatusNotFound,
	core.ErrRecapConflict:   http.StatusConflict,
}

// generationErrors maps the refusals of the generation operations to their status. A 422 answers a
// generation that outgrows every call its kind allows, a 429 one genai admits no call of, and a 503 one
// genai cannot take.
var generationErrors = httpf.ErrMap{
	core.ErrInvalidRequest:     http.StatusBadRequest,
	core.ErrProjectNotFound:    http.StatusNotFound,
	core.ErrBeatNotFound:       http.StatusNotFound,
	core.ErrSceneNotFound:      http.StatusNotFound,
	core.ErrLayerNotFound:      http.StatusNotFound,
	core.ErrGenerationNotFound: http.StatusNotFound,
	core.ErrContextTooLarge:    http.StatusUnprocessableEntity,
	core.ErrGenerationQuota:    http.StatusTooManyRequests,
	core.ErrGenaiUnavailable:   http.StatusServiceUnavailable,
}
