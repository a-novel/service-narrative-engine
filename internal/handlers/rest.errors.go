package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/picture"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
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

// instanceErrors maps the refusals of the instance, default instance and links operations to their status. A
// 422 answers a request that fails validation, a document over a cap included. A 409 answers a write that
// conflicts with the project's state: a document that changed since the caller read it, the delete of the
// default instance, a project that holds as many instances as it may, or one past its byte budget.
var instanceErrors = httpf.ErrMap{
	core.ErrInvalidRequest:     http.StatusUnprocessableEntity,
	core.ErrProjectNotFound:    http.StatusNotFound,
	core.ErrInstanceNotFound:   http.StatusNotFound,
	core.ErrInstanceStale:      http.StatusConflict,
	core.ErrInstanceDefault:    http.StatusConflict,
	core.ErrInstancesFull:      http.StatusConflict,
	core.ErrInstanceLinksStale: http.StatusConflict,
	core.ErrProjectBudget:      http.StatusConflict,
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
// 409: projectBytes for the bytes a project stores, projects for the projects a user owns, instances for the
// instances a project holds. Every handler whose service keeps one of these limits passes its error through
// it.
func tagLimit(err error) error {
	switch {
	case errors.Is(err, core.ErrProjectBudget):
		return httpf.WithTag(err, "limit", "projectBytes")
	case errors.Is(err, core.ErrProjectsFull):
		return httpf.WithTag(err, "limit", "projects")
	case errors.Is(err, core.ErrInstancesFull):
		return httpf.WithTag(err, "limit", "instances")
	default:
		return err
	}
}

// tagField names the field a [core.FieldError] refuses under the problem's invalidFields tag, beside the rule
// it broke, as golib names the fields a validator refuses. Every handler whose service checks a document
// passes its error through it.
func tagField(err error) error {
	refused, ok := errors.AsType[*core.FieldError](err)
	if !ok {
		return err
	}

	return httpf.WithTag(err, httpf.InvalidFieldsTag, map[string]string{refused.Path: refused.Rule})
}

// itemFields names the invalid fields of a one-item batch request as the single-item route's body names them,
// so its invalidFields stay those of its own request: plugin for Instances[0].Plugin, content for
// instances[0].content. Every single-item body is flat, so the last name of a field's path is its path. golib
// names the fields only when this tag is unset.
func itemFields(err error) error {
	if refused, ok := errors.AsType[*core.FieldError](err); ok {
		name := refused.Path[strings.LastIndex(refused.Path, ".")+1:]

		return httpf.WithTag(err, httpf.InvalidFieldsTag, map[string]string{name: refused.Rule})
	}

	fields, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}

	paths := make(map[string]string, len(fields))

	for _, field := range fields {
		name := field.StructField()
		paths[strings.ToLower(name[:1])+name[1:]] = field.Tag()
	}

	return httpf.WithTag(err, httpf.InvalidFieldsTag, paths)
}
