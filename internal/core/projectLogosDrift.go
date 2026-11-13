package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

// ProjectLogosDriftDao compares the logos a client holds with those of the caller's projects.
type ProjectLogosDriftDao interface {
	Exec(ctx context.Context, request *dao.ProjectLogosDriftRequest) ([]*dao.ProjectLogosDriftRow, error)
}

// ProjectLogosDriftRequest carries the logos a client holds.
type ProjectLogosDriftRequest struct {
	OwnerID uuid.UUID `validate:"required"`

	// Logos maps the ID of each project whose logo the client holds to the logo's checksum, as [Project.LogoChecksum]
	// carries it. A project the client holds without a logo stays out. It holds at most [ProjectsPerOwnerMax]
	// logos, and none when the client holds none.
	Logos map[string]string `validate:"required,max=100,dive,keys,uuid,endkeys,logochecksum"`
}

// ProjectLogosDriftResult is the drift between the logos a client holds and those of the caller's projects.
type ProjectLogosDriftResult struct {
	// Logos maps each project of the caller whose logo the request leaves out, or names with another checksum, to
	// the logo's checksum.
	Logos map[uuid.UUID]string
	// Removed are the IDs of the request whose project is not the caller's or holds no logo.
	Removed []uuid.UUID
}

// ProjectLogosDrift compares the logos a client holds with those of the caller's projects, by checksum, in one
// query. No logo's bytes leave the database.
type ProjectLogosDrift struct {
	dao ProjectLogosDriftDao
}

func NewProjectLogosDrift(dao ProjectLogosDriftDao) *ProjectLogosDrift {
	return &ProjectLogosDrift{dao: dao}
}

func (service *ProjectLogosDrift) Exec(
	ctx context.Context, request *ProjectLogosDriftRequest,
) (*ProjectLogosDriftResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectLogosDrift")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("logos.held", len(request.Logos)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Each logo of the owner's projects, and each held ID that names none, makes one row at most.
	rows, err := service.dao.Exec(ctx, &dao.ProjectLogosDriftRequest{
		OwnerID: request.OwnerID,
		Logos:   request.Logos,
		Limit:   ProjectsPerOwnerMax + len(request.Logos),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("compare project logos: %w", err))
	}

	result := &ProjectLogosDriftResult{Logos: make(map[uuid.UUID]string, len(rows)), Removed: []uuid.UUID{}}

	for _, row := range rows {
		if row.Checksum == nil {
			result.Removed = append(result.Removed, row.ProjectID)

			continue
		}

		result.Logos[row.ProjectID] = *row.Checksum
	}

	return result, nil
}
