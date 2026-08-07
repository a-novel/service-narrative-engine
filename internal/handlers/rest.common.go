package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

var restServiceErrors = httpf.ErrMap{
	ErrActorClaims:                    http.StatusUnauthorized,
	ErrAnonymousActor:                 http.StatusUnauthorized,
	core.ErrInvalidRequest:            http.StatusUnprocessableEntity,
	core.ErrProjectNotFound:           http.StatusNotFound,
	core.ErrGenerationNotFound:        http.StatusNotFound,
	core.ErrGenerationConflict:        http.StatusConflict,
	core.ErrGenerationOutputInvalid:   http.StatusBadGateway,
	core.ErrGenerationRefused:         http.StatusBadGateway,
	core.ErrGenerationResponseInvalid: http.StatusBadGateway,
	core.ErrGenerationStatusUnknown:   http.StatusBadGateway,
}

var restTransportErrors = httpf.ErrMap{
	ErrActorClaims:    http.StatusUnauthorized,
	ErrAnonymousActor: http.StatusUnauthorized,
	nil:               http.StatusBadRequest,
}

var errMultipleJSONDocuments = errors.New("expected one JSON document")

// RestIdeaValue is the static Idea content shared by create and version responses.
type RestIdeaValue struct {
	Title string `json:"title"`
	Genre string `json:"genre"`
	Seed  string `json:"seed"`
}

// RestIdeaVersion is one immutable Idea save.
type RestIdeaVersion struct {
	ID        uuid.UUID     `json:"id"`
	ProjectID uuid.UUID     `json:"projectID"`
	Value     RestIdeaValue `json:"value"`
	CreatedAt time.Time     `json:"createdAt"`
}

// RestStepValueVersion is one immutable opaque Step save.
type RestStepValueVersion struct {
	ID        uuid.UUID       `json:"id"`
	ProjectID uuid.UUID       `json:"projectID"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	CreatedAt time.Time       `json:"createdAt"`
}

// RestManuscriptVersion is one immutable static Manuscript save.
type RestManuscriptVersion struct {
	ID        uuid.UUID       `json:"id"`
	ProjectID uuid.UUID       `json:"projectID"`
	Value     json.RawMessage `json:"value"`
	CreatedAt time.Time       `json:"createdAt"`
}

// RestProject is the current convenience snapshot for one Project.
type RestProject struct {
	ID         uuid.UUID               `json:"id"`
	CreatedAt  time.Time               `json:"createdAt"`
	Idea       *RestIdeaVersion        `json:"idea"`
	StepValues []*RestStepValueVersion `json:"stepValues"`
	Manuscript *RestManuscriptVersion  `json:"manuscript"`
}

// RestGeneration is the provider-independent lifecycle and proposal contract.
type RestGeneration struct {
	ID          uuid.UUID             `json:"id"`
	Status      core.GenerationStatus `json:"status"`
	Attempt     int32                 `json:"attempt"`
	MaxAttempts int32                 `json:"maxAttempts"`
	Proposal    json.RawMessage       `json:"proposal"`
	Failure     *string               `json:"failure"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	SettledAt   *time.Time            `json:"settledAt"`
	ExpiresAt   *time.Time            `json:"expiresAt"`
}

// decodeRestJSON accepts exactly one JSON document and rejects unknown envelope fields.
func decodeRestJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}

	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode request body: %w", errMultipleJSONDocuments)
	}

	return nil
}

// restProjectIdentity resolves the authenticated Actor and Project path parameter.
func restProjectIdentity(r *http.Request) (*core.Actor, uuid.UUID, error) {
	actor, err := ActorFromContext(r.Context())
	if err != nil {
		return nil, uuid.Nil, err
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("parse Project ID: %w", err)
	}

	return actor, projectID, nil
}

// loadRestIdeaVersion removes ownership metadata from one public version response.
func loadRestIdeaVersion(idea *core.Idea) *RestIdeaVersion {
	if idea == nil {
		return nil
	}

	return &RestIdeaVersion{
		ID:        idea.VersionID,
		ProjectID: idea.ProjectID,
		Value: RestIdeaValue{
			Title: idea.Title,
			Genre: idea.Genre,
			Seed:  idea.Seed,
		},
		CreatedAt: idea.CreatedAt,
	}
}

// loadRestStepValueVersion preserves the opaque JSON value without decoding it.
func loadRestStepValueVersion(value *core.StepValue) *RestStepValueVersion {
	if value == nil {
		return nil
	}

	return &RestStepValueVersion{
		ID:        value.ID,
		ProjectID: value.ProjectID,
		Key:       value.Key,
		Value:     value.Value,
		CreatedAt: value.CreatedAt,
	}
}

// loadRestManuscriptVersion preserves the validated static document as JSON.
func loadRestManuscriptVersion(manuscript *core.Manuscript) *RestManuscriptVersion {
	if manuscript == nil {
		return nil
	}

	return &RestManuscriptVersion{
		ID:        manuscript.ID,
		ProjectID: manuscript.ProjectID,
		Value:     manuscript.Manuscript,
		CreatedAt: manuscript.CreatedAt,
	}
}

// loadRestGeneration keeps service-genai envelopes and provider errors private.
func loadRestGeneration(generation *core.Generation) *RestGeneration {
	if generation == nil {
		return nil
	}

	var failure *string
	if generation.Failure != "" {
		failure = &generation.Failure
	}

	return &RestGeneration{
		ID:          generation.ID,
		Status:      generation.Status,
		Attempt:     generation.Attempt,
		MaxAttempts: generation.MaxAttempts,
		Proposal:    generation.Proposal,
		Failure:     failure,
		CreatedAt:   generation.CreatedAt,
		UpdatedAt:   generation.UpdatedAt,
		SettledAt:   generation.SettledAt,
		ExpiresAt:   generation.ExpiresAt,
	}
}
