package handlers_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
)

func TestActorFromContext(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000042")

	testCases := []struct {
		name string

		claims *serviceauthentication.Claims

		expect    *core.Actor
		expectErr error
	}{
		{
			name:   "Success",
			claims: &serviceauthentication.Claims{UserID: &userID},
			expect: &core.Actor{UserID: userID},
		},
		{
			name:      "Error/Anonymous",
			claims:    &serviceauthentication.Claims{},
			expectErr: handlers.ErrAnonymousActor,
		},
		{
			name:      "Error/MissingClaims",
			expectErr: handlers.ErrActorClaims,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if testCase.claims != nil {
				ctx = serviceauthentication.SetClaimsContext(ctx, testCase.claims)
			}

			actor, err := handlers.ActorFromContext(ctx)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, actor)
		})
	}
}
