package core

import (
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	// ErrItemGetNotFound is returned when no item matches a get request.
	ErrItemGetNotFound = dao.ErrItemGetNotFound
	// ErrItemDeleteNotFound is returned when no item matches a delete request.
	ErrItemDeleteNotFound = dao.ErrItemDeleteNotFound
	// ErrItemUpdateNotFound is returned when no item matches an update request.
	ErrItemUpdateNotFound = dao.ErrItemUpdateNotFound
)
