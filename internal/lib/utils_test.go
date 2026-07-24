package lib_test

import (
	"os"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TestMain installs the W3C trace-context propagator before any test runs.
//
// otelhttp reads the global propagator at request time, and its default injects nothing, so the
// trace-context assertion in this package would pass over an empty header without this. Production
// reaches the same state through otel.Init, which points every preset — the disabled one
// included — at TraceContext.
//
// It runs once, ahead of the first test, so no parallel test observes the global mid-write.
func TestMain(m *testing.M) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	os.Exit(m.Run())
}
