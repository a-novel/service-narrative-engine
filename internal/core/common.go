package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

const (
	// ProjectBytesMax caps the UTF-8 bytes a project stores: its own fields, and every row it holds, which are
	// its instances and its links.
	ProjectBytesMax = 16 << 20
	// ProjectsPerOwnerMax caps the projects a user owns. Each holds at most [ProjectBytesMax], so an account
	// stores 1.6 GiB at most. A writer keeps a handful of novels, and 100 leaves room for drafts and copies.
	ProjectsPerOwnerMax = 100
)

var (
	// ErrProjectNotFound is returned when no project has the ID, or another user owns it.
	ErrProjectNotFound = dao.ErrProjectNotFound
	// ErrProjectLogoNotFound is returned when the project holds no logo.
	ErrProjectLogoNotFound = errors.New("project logo not found")
	// ErrProjectBudget is returned by a write that would bring a project past [ProjectBytesMax] stored bytes.
	ErrProjectBudget = errors.New("project budget reached")
	// ErrProjectsFull is returned by [ProjectCreate] when the user already owns [ProjectsPerOwnerMax]
	// projects.
	ErrProjectsFull = errors.New("projects full")
	// ErrInstanceNotFound is returned when the project holds no instance with the ID.
	ErrInstanceNotFound = dao.ErrInstanceNotFound
	// ErrInstanceStale is returned by [InstanceContentUpdate] when the instance changed since the caller read
	// it. The write changes nothing.
	ErrInstanceStale = dao.ErrInstanceContentUpdateStale
	// ErrInstanceDefault is returned by [InstanceDelete] when the instance is the default of its project.
	ErrInstanceDefault = dao.ErrInstanceDeleteDefault
	// ErrInstancesFull is returned by [InstancesCreate] when the project would hold more instances than it
	// may.
	ErrInstancesFull = errors.New("instances full")
	// ErrProjectLinksStale is returned by [ProjectLinksUpdate] when the links changed since the caller read
	// them. The write changes nothing.
	ErrProjectLinksStale = dao.ErrProjectLinksUpsertStale
	// ErrPluginVersionNotFound is returned when the registry holds no such version of the plugin.
	ErrPluginVersionNotFound = dao.ErrPluginVersionNotFound
	// ErrPluginModuleNotFound is returned when the registry holds no such version of the plugin, or the
	// version ships without the module.
	ErrPluginModuleNotFound = dao.ErrPluginModuleNotFound
	// ErrPluginMessagesNotFound is returned when the registry holds no such version of the plugin, or the
	// version is not translated in the language.
	ErrPluginMessagesNotFound = dao.ErrPluginMessagesNotFound
)

// ProjectOwnerDao checks that a user owns a project. A service that checks ownership without the project lock
// takes it for [checkProjectOwnership].
type ProjectOwnerDao interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// projectOwnershipOpts names the project to check and the user who claims it.
type projectOwnershipOpts struct {
	projectID uuid.UUID
	ownerID   uuid.UUID
}

// checkProjectOwnership returns [ErrProjectNotFound] unless the user owns the project, so another
// user's project reads as missing.
func checkProjectOwnership(ctx context.Context, daoOwner ProjectOwnerDao, opts projectOwnershipOpts) error {
	ctx, span := otel.Tracer().Start(ctx, "core.checkProjectOwnership")
	defer span.End()

	owned, err := daoOwner.Exec(ctx, &dao.ProjectOwnedRequest{ID: opts.projectID, OwnerID: opts.ownerID})
	if err != nil {
		return otel.ReportError(span, err)
	}

	if !owned {
		return otel.ReportError(span, ErrProjectNotFound)
	}

	return nil
}

// ProjectLockDao locks a project its owner names until the transaction ends, and stamps its edit. Every
// service that writes a project's content takes it for [lockOwnedProject].
type ProjectLockDao interface {
	Exec(ctx context.Context, request *dao.ProjectLockRequest) (bool, error)
}

// lockOwnedProject locks the project for the rest of the transaction, so its writes run one at a time,
// stamps the edit as the project's EditedAt, and returns [ErrProjectNotFound] unless the user owns it. One
// query does all three, and a write that fails later rolls the stamp back with it.
//
// The stamp is the instant the write asks for the lock. The write stamps its own records once it holds the
// lock, so a write that waited still stamps them later than the write before it did.
//
// Every write to the project's content calls it before its first write, as TestProjectWriteLock checks.
func lockOwnedProject(ctx context.Context, daoLock ProjectLockDao, opts projectOwnershipOpts) error {
	ctx, span := otel.Tracer().Start(ctx, "core.lockOwnedProject")
	defer span.End()

	request := &dao.ProjectLockRequest{ID: opts.projectID, OwnerID: opts.ownerID, Now: time.Now()}

	owned, err := daoLock.Exec(ctx, request)
	if err != nil {
		return otel.ReportError(span, err)
	}

	if !owned {
		return otel.ReportError(span, ErrProjectNotFound)
	}

	return nil
}

// ProjectBytesDao reads how many bytes a project stores. Every service that grows a project takes it for
// [keepProjectBudget].
type ProjectBytesDao interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// projectBudgetOpts names the project whose budget to keep.
type projectBudgetOpts struct {
	projectID uuid.UUID
}

// keepProjectBudget returns [ErrProjectBudget] when the project stores more than [ProjectBytesMax] bytes. A
// write runs it after its write, in the same transaction, so the count holds that write and a refusal
// rolls it back.
func keepProjectBudget(ctx context.Context, daoBytes ProjectBytesDao, opts projectBudgetOpts) error {
	ctx, span := otel.Tracer().Start(ctx, "core.keepProjectBudget")
	defer span.End()

	stored, err := daoBytes.Exec(ctx, &dao.ProjectBytesRequest{ID: opts.projectID})
	if err != nil {
		return otel.ReportError(span, err)
	}

	if stored.StoredBytes > ProjectBytesMax {
		return otel.ReportError(span, fmt.Errorf(
			"%w: %d bytes, over %d", ErrProjectBudget, stored.StoredBytes, ProjectBytesMax,
		))
	}

	return nil
}
