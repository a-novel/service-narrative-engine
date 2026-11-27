package handlers

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/samber/lo"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
)

// RestPluginVersionPublishService is the core operation [RestPluginVersionPublish] delegates to.
type RestPluginVersionPublishService interface {
	Exec(ctx context.Context, request *core.PluginVersionPublishRequest) error
}

// RestPluginVersionPublishLanguage holds the translations of a [RestPluginVersionPublishRequest] in one
// language.
type RestPluginVersionPublishLanguage struct {
	// Title names the plugin in the list of plugins.
	Title string `json:"title"`
	// Description says what the plugin does, in the list of plugins.
	Description string `json:"description"`
	// Messages holds the translations of the plugin's interface and of its schema, by key.
	Messages json.RawMessage `json:"messages"`
}

// RestPluginVersionPublishRequest is the JSON body the publish-version endpoint accepts: one whole version of
// a plugin.
type RestPluginVersionPublishRequest struct {
	Plugin  string `json:"plugin"`
	Version string `json:"version"`

	// Schema is the JSON Schema the content of an instance follows.
	Schema json.RawMessage `json:"schema"`
	// Exports holds the lists of linkable items an instance exposes, by name.
	Exports json.RawMessage `json:"exports"`
	// UI is the source of the JavaScript module that renders the plugin.
	UI string `json:"ui"`
	// Migration is the source of the JavaScript module that brings content from the previous version to this
	// one. Nil when the version needs none.
	Migration *string `json:"migration"`

	// Languages holds the translations of the version by language, a BCP 47 tag.
	Languages map[string]RestPluginVersionPublishLanguage `json:"languages"`

	// Proof is the token the API key of the request signed for the version, a compact JWT of type
	// "proof+jwt". Empty when no key authenticated the request.
	Proof string `json:"proof"`
}

// RestPluginVersionPublish is the HTTP handler for the publish-version endpoint. It responds 201 with no
// body and a Location naming the read of the version's manifest (RFC 9110 §15.3.2).
//
// The route verifies its caller, an API key, before this handler runs. The handler reads the ID of that key
// from the request's context, and no user. A refused proof answers 422 and names the proof member, as a
// member that breaks a validation rule is named.
type RestPluginVersionPublish struct {
	service RestPluginVersionPublishService
	logger  logging.Log
}

// decodeClosedJSON decodes the one JSON value of a body into destination, as [httpf.DecodeJSON] does, and
// refuses a member destination does not declare, which that function drops in silence.
//
// A version never changes once published. Stored without a member the registry does not know yet, it would
// lack that member for good, so the publication is refused instead.
func decodeClosedJSON(body io.Reader, destination any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(destination)
	if err != nil {
		return fmt.Errorf("decode JSON request body: %w", err)
	}

	// A body holding nothing past its first value ends in io.EOF.
	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode JSON request body: %w", cmp.Or(err, httpf.ErrJSONMultipleValues))
	}

	return nil
}

func NewRestPluginVersionPublish(
	service RestPluginVersionPublishService, logger logging.Log,
) *RestPluginVersionPublish {
	return &RestPluginVersionPublish{service: service, logger: logger}
}

func (handler *RestPluginVersionPublish) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.PluginVersionPublish")
	defer span.End()

	var request RestPluginVersionPublishRequest

	err := decodeClosedJSON(r.Body, &request)
	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, decodeErrors, err)

		return
	}

	// The key is absent for a request the route took without a token, as a local stack does.
	apiKeyID, _ := servicejsonkeys.RequestAPIKey(ctx)

	serviceRequest := &core.PluginVersionPublishRequest{
		Plugin:  request.Plugin,
		Version: request.Version,
		Schema:  request.Schema,
		Exports: request.Exports,
		UI:      []byte(request.UI),
		Languages: lo.MapValues(
			request.Languages,
			func(language RestPluginVersionPublishLanguage, _ string) core.PluginVersionPublishLanguage {
				return core.PluginVersionPublishLanguage(language)
			},
		),
		Proof:    request.Proof,
		APIKeyID: apiKeyID,
	}

	if request.Migration != nil {
		serviceRequest.Migration = []byte(*request.Migration)
	}

	err = handler.service.Exec(ctx, serviceRequest)
	if errors.Is(err, core.ErrPluginProof) {
		err = httpf.WithTag(err, httpf.InvalidFieldsTag, map[string]string{"proof": "pluginproof"})
	}

	if err != nil {
		httpf.HandleError(ctx, handler.logger, w, span, pluginErrors, tagField(err))

		return
	}

	location := url.Values{"plugin": {request.Plugin}, "version": {request.Version}}

	w.Header().Set("Location", "/v1/plugin/version?"+location.Encode())
	w.WriteHeader(http.StatusCreated)
}
