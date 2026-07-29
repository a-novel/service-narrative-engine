package core

import "github.com/google/uuid"

// Actor is the authenticated identity a request acts on behalf of.
// Core services use it for project-level authorization.
type Actor struct {
	// UserID is the authenticated user. Anonymous actors are rejected before
	// entering the service layer.
	UserID uuid.UUID `validate:"required"`
}
