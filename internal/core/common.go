package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

const (
	// ProjectBytesMax caps the UTF-8 bytes a project stores, over its scenes, cells, recaps and every other
	// large field. A 200k-word novel takes about 1.3 MB per manuscript layer, and a heavy project with
	// notes, recaps and a few branches stays under 8 MB, so the cap leaves it room to double.
	ProjectBytesMax = 16 << 20
	// ListPageBytesMax bounds the bytes a page of a list holds. It fits the largest single cell, 2 MiB of
	// content and 2 MiB of recap, so a page always holds one row.
	ListPageBytesMax = 4 << 20
	// ProjectsPerOwnerMax caps the projects a user owns. Each holds at most [ProjectBytesMax], so an account
	// stores 1.6 GiB at most. A writer keeps a handful of novels, and 100 leaves room for drafts and copies.
	ProjectsPerOwnerMax = 100
)

var (
	// ErrProjectNotFound is returned when no project has the ID, or another user owns it.
	ErrProjectNotFound = errors.New("project not found")
	// ErrBeatNotFound is returned when the project holds no beat with the ID.
	ErrBeatNotFound = errors.New("beat not found")
	// ErrSceneNotFound is returned when the project holds no scene with the ID.
	ErrSceneNotFound = errors.New("scene not found")
	// ErrLayerNotFound is returned when the project holds no layer with the ID.
	ErrLayerNotFound = errors.New("layer not found")
	// ErrPictureNotFound is returned when the project holds no such picture.
	ErrPictureNotFound = errors.New("picture not found")
	// ErrProjectFull is returned when a project already holds as many beats or layers as it may, a beat as
	// many scenes, or a write would bring a scene past [SceneContentMax] characters.
	ErrProjectFull = errors.New("project full")
	// ErrRequiredLayer is returned by a delete that would leave a project without a layer of a required
	// kind, such as the manuscript.
	ErrRequiredLayer = errors.New("required layer")
	// ErrProjectBudget is returned by a write that would bring a project past [ProjectBytesMax] stored bytes,
	// and by a whole-project read of a project already past it.
	ErrProjectBudget = errors.New("project budget reached")
	// ErrProjectsFull is returned by [ProjectCreate] when the user already owns [ProjectsPerOwnerMax]
	// projects.
	ErrProjectsFull = errors.New("projects full")
	// ErrGenerationNotFound is returned when genai holds no generation with the ID for the caller.
	ErrGenerationNotFound = errors.New("generation not found")
	// ErrGenerationQuota is returned by [GenerationSubmit], as a [GenerationQuotaError], when genai admits
	// none of a generation's calls.
	ErrGenerationQuota = errors.New("generation quota reached")
	// ErrGenaiUnavailable is returned when genai cannot take a call, such as during its downtime.
	ErrGenaiUnavailable = errors.New("genai unavailable")
	// ErrCellNotFound is returned when the project holds no cell where the scene and the layer cross.
	ErrCellNotFound = errors.New("cell not found")
	// ErrRecapConflict is returned by a recap write that read its source at a version older than the
	// stored recap's, or newer than the source's own.
	ErrRecapConflict = errors.New("recap conflict")
	// ErrGenerationNotStreamed is returned by [GenerationStream] for a call it cannot stream: one
	// submitted without streaming, or one the provider no longer streams. A read returns the call.
	ErrGenerationNotStreamed = errors.New("generation not streamed")
	// ErrStreamEnded is returned when the caller resumes a stream after its last event.
	ErrStreamEnded = errors.New("stream ended")
)

// projectOwnerDao checks that a user owns a project. Each service names its own interface, and every
// one satisfies this.
type projectOwnerDao interface {
	Exec(ctx context.Context, request *dao.ProjectOwnedRequest) (bool, error)
}

// projectOwnershipOpts names the project to check and the user who claims it.
type projectOwnershipOpts struct {
	projectID uuid.UUID
	ownerID   uuid.UUID
}

// checkProjectOwnership returns [ErrProjectNotFound] unless the user owns the project, so another
// user's project reads as missing.
func checkProjectOwnership(ctx context.Context, daoOwner projectOwnerDao, opts projectOwnershipOpts) error {
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

// projectLockDao locks a project its owner names until the transaction ends, and stamps its edit. Each service
// names its own interface, and every one satisfies this.
type projectLockDao interface {
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
func lockOwnedProject(ctx context.Context, daoLock projectLockDao, opts projectOwnershipOpts) error {
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

// projectBytesDao reads how many bytes a project stores. Each service names its own interface, and every
// one satisfies this.
type projectBytesDao interface {
	Exec(ctx context.Context, request *dao.ProjectBytesRequest) (*dao.ProjectBytes, error)
}

// projectBudgetOpts names the project whose budget to keep.
type projectBudgetOpts struct {
	projectID uuid.UUID
}

// keepProjectBudget returns [ErrProjectBudget] when the project stores more than [ProjectBytesMax] bytes. A
// write runs it after its write, in the same transaction, so the count holds that write and a refusal
// rolls it back. A whole-project read runs it before it reads.
func keepProjectBudget(ctx context.Context, daoBytes projectBytesDao, opts projectBudgetOpts) error {
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

// sceneLengthDao reads how many characters the cells of scenes hold. Each service names its own interface,
// and every one satisfies this.
type sceneLengthDao interface {
	Exec(ctx context.Context, request *dao.SceneLengthRequest) ([]*dao.SceneLength, error)
}

// checkSceneContent returns [ErrProjectFull] when one of the scenes holds more than [SceneContentMax]
// characters. A cell write runs it after its write, in the transaction that locked the project, so the count
// holds that write and every earlier one.
func checkSceneContent(ctx context.Context, daoLength sceneLengthDao, projectID uuid.UUID, sceneIDs []uuid.UUID) error {
	ctx, span := otel.Tracer().Start(ctx, "core.checkSceneContent")
	defer span.End()

	lengths, err := daoLength.Exec(ctx, &dao.SceneLengthRequest{ProjectID: projectID, IDs: sceneIDs})
	if err != nil {
		return otel.ReportError(span, err)
	}

	for _, scene := range lengths {
		if scene.Length > SceneContentMax {
			return otel.ReportError(span, fmt.Errorf(
				"%w: scene %s would hold %d characters", ErrProjectFull, scene.SceneID, scene.Length,
			))
		}
	}

	return nil
}

// newIDs generates count UUIDv7 identifiers.
func newIDs(ctx context.Context, count int) ([]uuid.UUID, error) {
	_, span := otel.Tracer().Start(ctx, "core.newIDs")
	defer span.End()

	ids := make([]uuid.UUID, count)

	for i := range ids {
		id, err := uuid.NewV7()
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("generate id: %w", err))
		}

		ids[i] = id
	}

	return ids, nil
}
