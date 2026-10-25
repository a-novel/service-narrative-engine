package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
)

// errAnonymousOwner reports an authenticated request whose claims name no user, such as one made
// with an anonymous session. Only a user owns projects.
var errAnonymousOwner = errors.New("anonymous caller owns no project")

// requestOwner returns the user an authenticated request acts for, from the claims the auth
// middleware verified.
func requestOwner(r *http.Request) (uuid.UUID, error) {
	claims, err := serviceauthentication.MustGetClaimsContext(r.Context())
	if err != nil {
		return uuid.Nil, fmt.Errorf("get claims: %w", err)
	}

	if claims.UserID == nil {
		return uuid.Nil, errAnonymousOwner
	}

	return *claims.UserID, nil
}
