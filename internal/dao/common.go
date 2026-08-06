package dao

import (
	"context"
	"errors"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

const contentVersionLimit = 25

// ErrProjectLockNotFound is returned when an owner-scoped Project cannot be locked for a write.
var ErrProjectLockNotFound = errors.New("project not found")

var errVersionedWriteRequiresTransaction = errors.New("versioned write requires a transaction")

func requireVersionedWriteTransaction(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.requireVersionedWriteTransaction")
	defer span.End()

	if !postgres.InTx(ctx) {
		return otel.ReportError(span, errVersionedWriteRequiresTransaction)
	}

	otel.ReportSuccessNoContent(span)

	return nil
}
