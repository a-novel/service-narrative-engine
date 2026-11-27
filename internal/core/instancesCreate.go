package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

// InstancesCreateDaoCount counts the instances the project holds.
type InstancesCreateDaoCount interface {
	Exec(ctx context.Context, request *dao.InstanceCountRequest) (*dao.InstanceCount, error)
}

// InstancesCreateDaoInsert inserts the instances, and returns them in the order of the request.
type InstancesCreateDaoInsert interface {
	Exec(ctx context.Context, request *dao.InstancesInsertRequest) ([]*dao.InstanceDescriptor, error)
}

// InstancesCreateDaoDefault makes an instance the default of the project.
type InstancesCreateDaoDefault interface {
	Exec(ctx context.Context, request *dao.ProjectDefaultUpdateRequest) (*dao.Project, error)
}

// InstancesCreateItem is one instance of an [InstancesCreateRequest].
type InstancesCreateItem struct {
	// Plugin is the name of the plugin, a DNS label (RFC 1035) as the registry names its plugins. The service
	// never checks that the registry holds it.
	Plugin string `validate:"required,dns_rfc1035_label"`
	// Version is the exact version of the plugin the content follows, as [parsePluginVersion] reads one,
	// [InstanceVersionMax] characters at most. The service never checks that the registry holds it.
	Version string `validate:"required,max=256,pluginversion"`
	// Name is the display name the writer gives the instance, [InstanceNameMax] characters at most. It may be
	// empty, and the client then shows a name of its own. A name of only whitespace is refused: it would show
	// as none, and no client would replace it.
	Name string `validate:"omitempty,notblank,max=80"`

	// Content is the JSON document the plugin edits, within the configured caps.
	Content json.RawMessage `validate:"required"`
}

// InstancesCreateRequest carries new instances of a project.
type InstancesCreateRequest struct {
	OwnerID   uuid.UUID `validate:"required"`
	ProjectID uuid.UUID `validate:"required"`

	// Instances lists the instances to create, as many at most as a project holds.
	Instances []InstancesCreateItem `validate:"required,min=1,dive"`
}

// InstancesCreate creates instances in a project and returns them in the order of the request, without their
// content. The first instance of a project becomes its default, which stamps the project's UpdatedAt. It
// creates all of them or none: none when the project would hold more instances than the configuration allows,
// which returns [ErrInstancesFull], or store more than [ProjectBytesMax] bytes.
type InstancesCreate struct {
	daoLock    ProjectLockDao
	daoCount   InstancesCreateDaoCount
	daoInsert  InstancesCreateDaoInsert
	daoDefault InstancesCreateDaoDefault
	daoBytes   ProjectBytesDao
	transactor transaction.Transactor
	caps       config.Caps
}

func NewInstancesCreate(
	daoLock ProjectLockDao,
	daoCount InstancesCreateDaoCount,
	daoInsert InstancesCreateDaoInsert,
	daoDefault InstancesCreateDaoDefault,
	daoBytes ProjectBytesDao,
	transactor transaction.Transactor,
	caps config.Caps,
) *InstancesCreate {
	return &InstancesCreate{
		daoLock:    daoLock,
		daoCount:   daoCount,
		daoInsert:  daoInsert,
		daoDefault: daoDefault,
		daoBytes:   daoBytes,
		transactor: transactor,
		caps:       caps,
	}
}

func (service *InstancesCreate) Exec(
	ctx context.Context, request *InstancesCreateRequest,
) ([]*InstanceDescriptor, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.InstancesCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("project.id", request.ProjectID.String()),
		attribute.String("project.owner_id", request.OwnerID.String()),
		attribute.Int("instances.count", len(request.Instances)),
	)

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// A batch larger than a project can hold fits none, whatever the project holds.
	if len(request.Instances) > service.caps.InstancesPerProject {
		return nil, otel.ReportError(span, &FieldError{Path: "instances", Rule: ruleMax})
	}

	for index, instance := range request.Instances {
		path := "instances[" + strconv.Itoa(index) + "].content"

		err = checkDocument(path, instance.Content, service.caps.ContentBytes, service.caps.DocumentDepth)
		if err != nil {
			return nil, otel.ReportError(span, err)
		}
	}

	var entities []*dao.InstanceDescriptor

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		err := lockOwnedProject(ctx, service.daoLock, projectOwnershipOpts{
			projectID: request.ProjectID,
			ownerID:   request.OwnerID,
		})
		if err != nil {
			return fmt.Errorf("lock project: %w", err)
		}

		// The lock holds the count until the insert commits, so two batches cannot both pass the cap.
		held, err := service.daoCount.Exec(ctx, &dao.InstanceCountRequest{ProjectID: request.ProjectID})
		if err != nil {
			return fmt.Errorf("count instances: %w", err)
		}

		if held.Count+len(request.Instances) > service.caps.InstancesPerProject {
			return fmt.Errorf(
				"%w: %d instances, and %d more", ErrInstancesFull, held.Count, len(request.Instances),
			)
		}

		// The stamp and the IDs are taken together once the lock is held: an ID carries the instant it was
		// made at, and a batch that waited stamps its instances later than the write before it did.
		now := time.Now()
		items := lo.Map(request.Instances, func(instance InstancesCreateItem, _ int) dao.InstancesInsertItem {
			return dao.InstancesInsertItem{
				// uuid.NewV7 fails only when crypto/rand does, which Go documents as never happening on
				// supported platforms.
				ID:      uuid.Must(uuid.NewV7()),
				Plugin:  instance.Plugin,
				Version: instance.Version,
				Name:    instance.Name,
				Content: instance.Content,
			}
		})

		entities, err = service.daoInsert.Exec(ctx, &dao.InstancesInsertRequest{
			ProjectID: request.ProjectID,
			Instances: items,
			Now:       now,
		})
		if err != nil {
			return fmt.Errorf("insert instances: %w", err)
		}

		// A default is never unset and its instance is never deleted, so a project has none exactly when it
		// holds no instance. Its first instance becomes it.
		if held.Count == 0 {
			_, err = service.daoDefault.Exec(ctx, &dao.ProjectDefaultUpdateRequest{
				ProjectID:  request.ProjectID,
				InstanceID: items[0].ID,
				Now:        now,
			})
			if err != nil {
				return fmt.Errorf("update project default: %w", err)
			}
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

	return lo.Map(entities, func(entity *dao.InstanceDescriptor, _ int) *InstanceDescriptor {
		return newInstanceDescriptor(entity)
	}), nil
}
