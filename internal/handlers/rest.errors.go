package handlers

import (
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

// ownerErrors maps a request that acts for no user to 403: one whose claims name none, such as an anonymous
// session's (serviceauthentication.ErrAnonymousCaller), or one that carries no claims.
var ownerErrors = httpf.ErrMap{nil: http.StatusForbidden}

// decodeErrors maps a request whose body or query cannot be read or parsed to its status.
var decodeErrors = httpf.ErrMap{nil: http.StatusBadRequest}

// projectErrors maps the refusals of the project operations to their status. A 422 answers a request that
// fails validation. A 409 answers a write that conflicts with the caller's state: a project past its byte
// budget, or a user who owns as many projects as they may.
var projectErrors = httpf.ErrMap{
	core.ErrInvalidRequest:  http.StatusUnprocessableEntity,
	core.ErrProjectNotFound: http.StatusNotFound,
	core.ErrProjectBudget:   http.StatusConflict,
	core.ErrProjectsFull:    http.StatusConflict,
}

// logoErrors maps the refusals of the logo operations to their status. A 422 answers a request that fails
// validation, and a logo the contract refuses. A refused logo carries its reason as the problem's reason tag.
var logoErrors = httpf.ErrMap{
	core.ErrInvalidRequest:      http.StatusUnprocessableEntity,
	core.ErrProjectNotFound:     http.StatusNotFound,
	core.ErrProjectLogoNotFound: http.StatusNotFound,
	picture.ErrRefused:          http.StatusUnprocessableEntity,
}

// pluginErrors maps the refusals of the plugin registry's reads to their status. A 422 answers a request that
// fails validation. A 404 answers a version the registry lacks, a module the version ships without, and a
// language it is not translated in.
var pluginErrors = httpf.ErrMap{
	core.ErrInvalidRequest:         http.StatusUnprocessableEntity,
	core.ErrPluginVersionNotFound:  http.StatusNotFound,
	core.ErrPluginModuleNotFound:   http.StatusNotFound,
	core.ErrPluginMessagesNotFound: http.StatusNotFound,
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
