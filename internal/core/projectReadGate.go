package core

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"runtime/debug"

	"github.com/a-novel-kit/golib/otel"
)

// ProjectReadGate bounds the reads of a whole project an instance runs at once: a manuscript export, or a
// generation plan, which reads a summary of each cell and the cells it may show. Each holds up to
// [ProjectBytesMax] in memory, several times over once decoded, a plan only for a check of every scene. The
// gate sizes their sum to the memory the instance has, and a read past the bound waits for a slot.
type ProjectReadGate struct {
	slots chan struct{}
}

// projectReadAmplification is how many times over a manuscript export holds the bytes it reads, once decoded and
// rendered. A generation plan reads a summary of each cell, then only the cells it may show, every one only for a
// check of every scene, and keeps no copy of them undecoded: the same bound covers it.
const projectReadAmplification = 8

// NewProjectReadGate returns a gate that runs size reads at once. A size under 1 sizes the gate to the memory
// the instance has: the reads take half of GOMEMLIMIT, at [ProjectBytesMax] times projectReadAmplification
// each, so 4 reads at 1 GiB, and 1 at least. Without a memory limit, it runs as many reads as CPUs, which they
// keep busy decoding.
func NewProjectReadGate(size int) *ProjectReadGate {
	if size < 1 {
		size = projectReadSlots()
	}

	return &ProjectReadGate{slots: make(chan struct{}, size)}
}

// projectReadSlots sizes a gate to the memory limit. debug.SetMemoryLimit with a negative limit reads the limit
// without changing it, and reports math.MaxInt64 when GOMEMLIMIT is unset.
func projectReadSlots() int {
	limit := debug.SetMemoryLimit(-1)
	if limit == math.MaxInt64 {
		return runtime.GOMAXPROCS(0)
	}

	return int(max(limit/2/(ProjectBytesMax*projectReadAmplification), 1))
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
