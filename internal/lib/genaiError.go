package lib

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func normalizeGenAIError(err error, code codes.Code, sentinel error) error {
	if status.Code(err) == code {
		return errors.Join(sentinel, err)
	}

	return err
}
