package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ownerErrors maps a request that names no user to its status.
var ownerErrors = httpf.ErrMap{nil: http.StatusForbidden}

// composerErrors maps the refusals of the Scenes Composer operations to their status. A client
// reads the data again after a 409, and stops adding after a 422.
var composerErrors = httpf.ErrMap{
	core.ErrInvalidRequest:   http.StatusBadRequest,
	core.ErrProjectNotFound:  http.StatusNotFound,
	core.ErrSceneNotFound:    http.StatusNotFound,
	core.ErrLayerNotFound:    http.StatusNotFound,
	core.ErrCellNotFound:     http.StatusNotFound,
	core.ErrRevisionConflict: http.StatusConflict,
	core.ErrProjectFull:      http.StatusUnprocessableEntity,
	core.ErrManuscriptLayer:  http.StatusUnprocessableEntity,
}
