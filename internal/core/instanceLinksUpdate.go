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

// linksDepth is how deep the links nest: a list of links, each holding its destination.
const linksDepth = 3

// InstanceLinksUpdateDao writes the links of the instance, only while they still hold the UpdatedAt the request
// names.
type InstanceLinksUpdateDao interface {
	Exec(ctx context.Context, request *dao.InstanceLinksUpdateRequest) (*dao.InstanceLinksUpdateRow, error)
}

// InstanceLinksUpdateRequest carries the new links of an instance.
type InstanceLinksUpdateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`
	ID        uuid.UUID `validate:"required"`

	// Links replace the links the instance stores, within the configured caps. An empty list clears them, and
	// a nil list fails validation, so an omission never wipes them.
	Links []Link `validate:"required,dive"`

	// UpdatedAt is the UpdatedAt the links carried when the caller read them, to the microsecond: nil for an
	// instance that never wrote its links. The first write names none, as every later write names the one
	// before it.
	UpdatedAt *time.Time
}

// InstanceLinksUpdate replaces the static links an instance stores in one write, and returns when it wrote
// them. The write applies only while the links still carry the UpdatedAt the request names: it merges nothing,
// and returns [ErrInstanceLinksStale] once another write changed them. The links and the content of an instance
// are stamped apart, so neither write makes the other stale. It writes nothing when the project would store more
// than [ProjectBytesMax] bytes.
//
// The service checks the shape and the size of the links. It never checks that an instance or an item they
// name exists: the client resolves them, and prunes the ones left dangling.
type InstanceLinksUpdate struct {
	daoLock    ProjectLockDao
	dao        InstanceLinksUpdateDao
	daoBytes   ProjectBytesDao
	transactor transaction.Transactor
	caps       config.Caps
}

func NewInstanceLinksUpdate(
	daoLock ProjectLockDao,
	dao InstanceLinksUpdateDao,
	daoBytes ProjectBytesDao,
	transactor transaction.Transactor,
	caps config.Caps,
) *InstanceLinksUpdate {
	return &InstanceLinksUpdate{daoLock: daoLock, dao: dao, daoBytes: daoBytes, transactor: transactor, caps: caps}
}

func (service *InstanceLinksUpdate) Exec(
	ctx context.Context, request *InstanceLinksUpdateRequest,
) (*WrittenInstanceLinks, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstanceLinksUpdate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.String("instance.id", request.ID.String()),
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

	var entity *dao.InstanceLinksUpdateRow

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		entity, err = service.dao.Exec(ctx, &dao.InstanceLinksUpdateRequest{
			ProjectID: request.ProjectID,
			ID:        request.ID,
			Links:     links,
			UpdatedAt: request.UpdatedAt,
			Now:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("update links: %w", err)
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

	return &WrittenInstanceLinks{UpdatedAt: entity.UpdatedAt}, nil
}

// encode returns the links as the JSON document the instance stores, or a [FieldError] for the first cap they
// pass.
func (service *InstanceLinksUpdate) encode(links []Link) (json.RawMessage, error) {
	if len(links) > service.caps.Links {
		return nil, &FieldError{Path: "links", Rule: ruleMax}
	}

	for index, link := range links {
		path := "links[" + strconv.Itoa(index) + "]."

		switch {
		case utf8.RuneCountInString(link.Source) > service.caps.LinkTargetLength:
			return nil, &FieldError{Path: path + "source", Rule: ruleMax}
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
