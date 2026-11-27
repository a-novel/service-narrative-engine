package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// linksDepth is how deep the links nest: a list of links, each holding its two ends.
const linksDepth = 3

// ProjectLinksUpdateDao writes the links of the project, only while they still hold the UpdatedAt the request
// names.
type ProjectLinksUpdateDao interface {
	Exec(ctx context.Context, request *dao.ProjectLinksUpsertRequest) (*dao.ProjectLinksUpsertRow, error)
}

// ProjectLinksUpdateRequest carries the new links of a project.
type ProjectLinksUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Links replace the links of the project, within the configured caps. An empty list clears them, and a
	// nil list fails validation, so an omission never wipes them.
	Links []Link `validate:"required,dive"`

	// UpdatedAt is the UpdatedAt the links carried when the caller read them, to the microsecond: nil when the
	// project had never written its links. The first write of a project applies whatever it names.
	UpdatedAt *time.Time
}

// ProjectLinksUpdate replaces the links of a project in one write, and returns when it wrote them. The write
// applies only while the links still carry the UpdatedAt the request names: it merges nothing, and returns
// [ErrProjectLinksStale] once another write changed them. It writes nothing when the project would store more
// than [ProjectBytesMax] bytes.
//
// The service checks the shape and the size of the links. It never checks that an instance or an item they
// name exists: the client resolves them, and prunes the ones left dangling.
type ProjectLinksUpdate struct {
	daoLock    ProjectLockDao
	dao        ProjectLinksUpdateDao
	daoBytes   ProjectBytesDao
	transactor transaction.Transactor
	caps       config.Caps
}

func NewProjectLinksUpdate(
	daoLock ProjectLockDao,
	dao ProjectLinksUpdateDao,
	daoBytes ProjectBytesDao,
	transactor transaction.Transactor,
	caps config.Caps,
) *ProjectLinksUpdate {
	return &ProjectLinksUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor, caps: caps}
}

func (service *ProjectLinksUpdate) Exec(
	ctx context.Context, request *ProjectLinksUpdateRequest,
) (*WrittenProjectLinks, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectLinksUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("links.count", len(request.Links)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	links, err := service.encode(request.Links)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var entity *dao.ProjectLinksUpsertRow

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err = service.dao.Exec(ctx, &dao.ProjectLinksUpsertRequest{
			ProjectID: request.ProjectID,
			Links:     links,
			UpdatedAt: request.UpdatedAt,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("upsert links: %w", err)
		}

		err = keepProjectBudget(ctx, service.daoBytes, projectBudgetOpts{projectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("keep project budget: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return &WrittenProjectLinks{UpdatedAt: entity.UpdatedAt}, nil
}

// encode returns the links as the JSON document the project stores, or a [FieldError] for the first cap they
// pass.
func (service *ProjectLinksUpdate) encode(links []Link) (json.RawMessage, error) {
	if len(links) > service.caps.Links {
		return nil, &FieldError{Path: "links", Rule: ruleMax}
	}

	for index, link := range links {
		path := "links[" + strconv.Itoa(index) + "]."

		switch {
		case utf8.RuneCountInString(link.Source.Target) > service.caps.LinkTargetLength:
			return nil, &FieldError{Path: path + "source.target", Rule: ruleMax}
		case utf8.RuneCountInString(link.Destination.Target) > service.caps.LinkTargetLength:
			return nil, &FieldError{Path: path + "destination.target", Rule: ruleMax}
		case len(link.Relationship) > service.caps.LinkRelationshipBytes:
			return nil, &FieldError{Path: path + "relationship", Rule: ruleMaxBytes}
		}
	}

	encoded, err := json.Marshal(links)
	if err != nil {
		return nil, fmt.Errorf("encode links: %w", err)
	}

	// The document check holds the links to their size, and refuses a NUL character in any of their texts.
	err = checkDocument("links", encoded, service.caps.LinksBytes, linksDepth)
	if err != nil {
		return nil, err
	}

	return encoded, nil
}
