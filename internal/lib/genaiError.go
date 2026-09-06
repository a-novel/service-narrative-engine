package lib

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func normalizeGenAIError(action string, err error, sentinels map[codes.Code]error) error {
	if err == nil {
		return nil
	}

	if sentinel, ok := sentinels[status.Code(err)]; ok {
		return fmt.Errorf("%w: %w", sentinel, err)
	}

	return fmt.Errorf("%s: %w", action, err)
}
