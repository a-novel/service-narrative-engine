package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var errIdeaInsertMissing = errors.New("idea insert returned no entity")

// IdeaCreateDao persists Ideas created by [IdeaCreate].
type IdeaCreateDao interface {
	Exec(ctx context.Context, request *dao.IdeaInsertRequest) (*dao.Idea, error)
}

// IdeaCreateRequest carries the authenticated owner and complete typed Idea content.
type IdeaCreateRequest struct {
	Actor Actor `validate:"actor"`
	Seed  string
	Genre string
	Title string
}

// IdeaCreate validates and persists a new Project with its first Idea Version.
type IdeaCreate struct {
	dao IdeaCreateDao
}

// NewIdeaCreate creates an Idea service.
func NewIdeaCreate(ideaDao IdeaCreateDao) *IdeaCreate {
	return &IdeaCreate{dao: ideaDao}
}

// Exec creates an Idea owned by the authenticated actor.
func (service *IdeaCreate) Exec(ctx context.Context, request *IdeaCreateRequest) (*Idea, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.IdeaCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = validateIdeaContent(request.Seed, request.Genre, request.Title)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(attribute.String("idea.owner_id", request.Actor.UserID.String()))

	entity, err := service.dao.Exec(ctx, &dao.IdeaInsertRequest{
		ProjectID: uuid.Must(uuid.NewV7()),
		VersionID: uuid.Must(uuid.NewV7()),
		OwnerID:   request.Actor.UserID,
		Seed:      request.Seed,
		Genre:     request.Genre,
		Title:     request.Title,
		Now:       time.Now(),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert idea: %w", err))
	}

	if entity == nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert idea: %w", errIdeaInsertMissing))
	}

	span.SetAttributes(
		attribute.String("project.id", entity.ProjectID.String()),
		attribute.String("idea.version_id", entity.VersionID.String()),
	)

	return otel.ReportSuccess(span, &Idea{
		ProjectID:        entity.ProjectID,
		VersionID:        entity.VersionID,
		OwnerID:          entity.OwnerID,
		Seed:             entity.Seed,
		Genre:            entity.Genre,
		Title:            entity.Title,
		ProjectCreatedAt: entity.ProjectCreatedAt,
		CreatedAt:        entity.CreatedAt,
	}), nil
}
