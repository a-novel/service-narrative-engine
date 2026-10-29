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
