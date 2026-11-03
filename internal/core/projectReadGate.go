package core

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
)

// ProjectReadGate bounds the reads of a whole project an instance runs at once, such as a manuscript
// export or the snapshot a generation plans from. Each holds up to [ProjectBytesMax] in memory, several
// times over once decoded, so the gate sizes their sum to the memory the instance has. A read past the
// bound waits for a slot.
type ProjectReadGate struct {
	slots chan struct{}
}

// NewProjectReadGate returns a gate that runs size reads at once, at least 1.
func NewProjectReadGate(size int) *ProjectReadGate {
	return &ProjectReadGate{slots: make(chan struct{}, max(size, 1))}
}

// Run runs read once a slot is free, and frees the slot when read returns. When ctx ends first, it returns
// the context's error without running read.
func (gate *ProjectReadGate) Run(ctx context.Context, read func(ctx context.Context) error) error {
	ctx, span := otel.Tracer().Start(ctx, "core.ProjectReadGate")
	defer span.End()

	select {
	case gate.slots <- struct{}{}:
	case <-ctx.Done():
		return otel.ReportError(span, fmt.Errorf("wait for a project read: %w", ctx.Err()))
	}

	defer func() { <-gate.slots }()

	err := read(ctx)
	if err != nil {
		return otel.ReportError(span, err)
	}

	return nil
}
