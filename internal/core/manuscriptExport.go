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

// ManuscriptExportDaoOwner checks that the user owns the project.
type ManuscriptExportDaoOwner interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// ManuscriptExportDaoKind reads the kind of the layer to export.
type ManuscriptExportDaoKind interface {
	Exec(ctx context.Context, request *dao.LayerKindRequest) ([]*dao.LayerKind, error)
}

// ManuscriptExportDaoBytes reads how many bytes the project stores.
type ManuscriptExportDaoBytes interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// ManuscriptExportDaoProject reads the project, for its title and language.
type ManuscriptExportDaoProject interface {
	Exec(ctx context.Context, request *dao.ProjectSelectRequest) (*dao.Project, error)
}

// ManuscriptExportDao reads every beat and scene of the project, each scene with its cell in the layer.
type ManuscriptExportDao interface {
	Exec(ctx context.Context, request *dao.ManuscriptSelectRequest) ([]*dao.ManuscriptRow, error)
}

// ManuscriptExportRequest names the manuscript layer to export.
type ManuscriptExportRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	// LayerID is the manuscript layer to export. A project may hold several, such as a branch of the story.
	LayerID uuid.UUID `validate:"required"`
}

// ManuscriptExport reads the text a manuscript layer holds across every scene of the project, with the
// beats the scenes belong to. A layer the project does not hold, or of another kind, answers
// [ErrLayerNotFound], and a project past [ProjectBytesMax] stored bytes [ErrProjectBudget]. The read and
// the build wait for a slot of the gate, which they share with every other read of a whole project.
type ManuscriptExport struct {
	daoOwner   ManuscriptExportDaoOwner
	daoKind    ManuscriptExportDaoKind
	daoBytes   ManuscriptExportDaoBytes
	daoProject ManuscriptExportDaoProject
	dao        ManuscriptExportDao
	gate       *ProjectReadGate
}

func NewManuscriptExport(
	daoOwner ManuscriptExportDaoOwner,
	daoKind ManuscriptExportDaoKind,
	daoBytes ManuscriptExportDaoBytes,
	daoProject ManuscriptExportDaoProject,
	dao ManuscriptExportDao,
	gate *ProjectReadGate,
) *ManuscriptExport {
	return &ManuscriptExport{
		daoOwner: daoOwner, daoKind: daoKind, daoBytes: daoBytes, daoProject: daoProject, dao: dao, gate: gate,
	}
}

func (service *ManuscriptExport) Exec(ctx context.Context, request *ManuscriptExportRequest) (*Manuscript, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ManuscriptExport")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("layer.id", request.LayerID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkProjectOwnership(ctx, service.daoOwner, projectOwnershipOpts{
		projectID: request.ProjectID,
		ownerID:   request.OwnerID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check owner: %w", err))
	}

	kinds, err := service.daoKind.Exec(ctx, &dao.LayerKindRequest{
		ProjectID: request.ProjectID, IDs: []uuid.UUID{request.LayerID},
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read layer kind: %w", err))
	}

	if len(kinds) == 0 || kinds[0].Kind != LayerKindManuscript {
		return nil, otel.ReportError(span, ErrLayerNotFound)
	}

	var manuscript *Manuscript

	err = service.gate.Run(ctx, func(ctx context.Context) error {
		err := keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		project, err := service.daoProject.Exec(ctx, &dao.ProjectSelectRequest{ID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("select project: %w", err)
		}

		rows, err := service.dao.Exec(ctx, &dao.ManuscriptSelectRequest{
			ProjectID: request.ProjectID, LayerID: request.LayerID,
		})
		if err != nil {
			return fmt.Errorf("select manuscript: %w", err)
		}

		manuscript, err = newManuscript(project, rows)

		return err
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("read manuscript: %w", err))
	}

	return manuscript, nil
}
