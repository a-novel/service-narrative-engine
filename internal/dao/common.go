package dao

import (
	"context"
	"errors"

	"github.com/a-novel-kit/golib/postgres"
)

const contentVersionLimit = 25

// ErrIdeaLockNotFound is returned when an owner-scoped Idea root cannot be locked for a write.
var ErrIdeaLockNotFound = errors.New("idea not found")

var errVersionedWriteRequiresTransaction = errors.New("versioned write requires a transaction")

func requireVersionedWriteTransaction(ctx context.Context) error {
	if !postgres.InTx(ctx) {
		return errVersionedWriteRequiresTransaction
	}

	return nil
}
