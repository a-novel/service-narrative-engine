package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

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

// rolesGranting returns the roles that grant a permission, on their own or through the roles they inherit.
func rolesGranting(permissions serviceauthentication.Permissions, permission string) map[string]bool {
	granting := map[string]bool{}

	for name := range permissions.Roles {
		stack, seen := []string{name}, map[string]bool{}

		for len(stack) > 0 && !granting[name] {
			role := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if seen[role] {
				continue
			}

			seen[role] = true
			granting[name] = slices.Contains(permissions.Roles[role].Permissions, permission)
			stack = append(stack, permissions.Roles[role].Inherits...)
		}
	}

	return granting
}

// requestGranted reports whether the claims of a request carry one of the granting roles.
func requestGranted(r *http.Request, granting map[string]bool) bool {
	claims, err := serviceauthentication.GetClaimsContext(r.Context())

	return err == nil && slices.ContainsFunc(claims.Roles, func(role string) bool { return granting[role] })
}
