package core

import "github.com/google/uuid"

// Actor is the authenticated identity a request acts on behalf of.
// Authorization remains at the handler boundary and is not carried here.
type Actor struct {
	// UserID is the authenticated user. Anonymous actors are rejected before
	// entering the service layer.
	UserID uuid.UUID `validate:"required"`
}
